package product

import (
	"context"
	"github.com/LeHNam/wao-api/helpers/utils"
	"github.com/LeHNam/wao-api/services/database"
	"github.com/LeHNam/wao-api/services/websocket"
	"time"

	svCtx "github.com/LeHNam/wao-api/context"
	"github.com/LeHNam/wao-api/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ProductServer struct {
	sc        *svCtx.ServiceContext
	wsService *websocket.WebSocketService
}

func NewProductServer(sc *svCtx.ServiceContext, wsService *websocket.WebSocketService) *ProductServer {
	return &ProductServer{
		sc:        sc,
		wsService: wsService,
	}
}

func (s *ProductServer) GetProduct(ctx context.Context, request GetProductRequestObject) (GetProductResponseObject, error) {
	cond := map[string]any{}

	userCtx := utils.GetUserFromContext(ctx)
	if request.Params.Search != nil {
		cond["OR"] = []map[string]any{
			{"products.name LIKE": "%" + *request.Params.Search + "%"},
			{"products.code LIKE": "%" + *request.Params.Search + "%"},
		}
	}

	sort := request.Params.Sort
	page := request.Params.Page
	limit := request.Params.Limit
	offset := (page - 1) * limit

	preloads := []database.PreloadData{
		{Field: "Options"},
	}

	// Initialize empty joins array for SQL JOIN clauses
	joins := []string{}
	if userCtx != nil && userCtx.Role != "BUYER" {
		preloads = []database.PreloadData{
			{Field: "Options", Args: []interface{}{"quantity > 0 AND PRICE > 0"}},
		}
	}
	products, err := s.sc.ProductRepo.FindWithJoinAndPreload(ctx, cond, []string{}, limit, offset, sort, joins, preloads)
	total, _ := s.sc.ProductRepo.CountWithJoin(ctx, cond, joins)
	if err != nil {
		return GetProduct400JSONResponse{
			Message: "failed to get list of products",
		}, nil
	}

	items := make([]Product, 0, len(products))
	for _, p := range products {
		options := make([]ProductOption, 0, len(p.Options))
		for _, po := range p.Options {
			options = append(options, ProductOption{
				Id:       po.ID.String(),
				Name:     po.Name,
				Code:     po.Code,
				Quantity: po.Quantity,
				Price:    float32(po.Price),
			})
		}
		items = append(items, Product{
			Id:      p.ID.String(),
			Name:    p.Name,
			Code:    p.Code,
			Img:     p.Img,
			Options: options,
		})

	}

	res := GetProduct200JSONResponse{
		Items: items,
		Limit: limit,
		Page:  page,
		Pages: (int(total) + limit - 1) / limit,
		Total: int(total),
	}

	return res, nil
}

func (s *ProductServer) PostProduct(ctx context.Context, request PostProductRequestObject) (PostProductResponseObject, error) {

	options := make([]models.ProductOption, 0, len(request.Body.Options))
	for _, o := range request.Body.Options {
		options = append(options, models.ProductOption{
			ID:       uuid.New(),
			Name:     o.Name,
			Code:     o.Code,
			Quantity: o.Quantity,
			Price:    float64(o.Price),
		})
	}

	productModel := &models.Product{
		ID:        uuid.New(),
		Name:      request.Body.Name,
		Code:      request.Body.Code,
		Img:       request.Body.Img,
		Options:   options,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := s.sc.ProductRepo.Create(ctx, productModel)

	if err != nil {
		return PostProduct400JSONResponse{
			Message: "failed to create product",
		}, nil
	}
	message := map[string]any{
		"event": "product_created",
		"data":  productModel,
	}
	_ = s.wsService.Broadcast(message)

	return PostProduct201JSONResponse{
		Message: nil,
		Data: Product{
			Id:   productModel.ID.String(),
			Name: productModel.Name,
		},
	}, nil
}

func (s *ProductServer) DeleteProductId(ctx context.Context, request DeleteProductIdRequestObject) (DeleteProductIdResponseObject, error) {
	id, err := uuid.Parse(request.Id)
	if err != nil {
		return DeleteProductId404JSONResponse{
			Message: "id not valid",
		}, nil
	}
	err = s.sc.ProductRepo.Delete(ctx, id)
	if err != nil {
		s.sc.Log.Error("failed to delete product", zap.Error(err))
		return DeleteProductId404JSONResponse{
			Message: "delete product failed",
		}, nil
	}
	return DeleteProductId204Response{}, nil
}

func (s *ProductServer) GetProductId(ctx context.Context, request GetProductIdRequestObject) (GetProductIdResponseObject, error) {
	id, err := uuid.Parse(request.Id)
	if err != nil {
		return GetProductId404JSONResponse{
			Message: "id not valid",
		}, nil
	}

	product, err := s.sc.ProductRepo.First(ctx, id)
	if err != nil {
		return GetProductId404JSONResponse{
			Message: "id not found",
		}, nil
	}

	productOptions, err := s.sc.ProductOptionRepo.Find(ctx, map[string]any{"product_id": id}, []string{}, 0, 0, nil)
	options := make([]ProductOption, len(productOptions))
	for i, o := range productOptions {
		options[i] = ProductOption{
			Id:       o.ID.String(),
			Name:     o.Name,
			Code:     o.Code,
			Quantity: o.Quantity,
			Price:    float32(o.Price),
		}
	}

	return GetProductId200JSONResponse{
		Message: nil,
		Data: Product{
			Id:      product.ID.String(),
			Name:    product.Name,
			Code:    product.Code,
			Img:     product.Img,
			Options: options,
		},
	}, nil
}

func (s *ProductServer) PutProductId(ctx context.Context, request PutProductIdRequestObject) (PutProductIdResponseObject, error) {
	id, err := uuid.Parse(request.Id)
	if err != nil {
		return PutProductId404JSONResponse{
			Message: "id not valid",
		}, nil
	}
	tx := s.sc.DB.Begin().WithContext(ctx)
	if tx.Error != nil {
		s.sc.Log.Error(tx.Error.Error())
		return PutProductId404JSONResponse{
			Message: "Create DB transaction failed",
		}, nil
	}
	defer tx.Rollback()

	if request.Body.Options != nil {
		// Delete existing options for the product
		err = s.sc.ProductOptionRepo.WithTx(tx).DeleteWhere(ctx, map[string]any{"product_id": id})
		if err != nil {
			s.sc.Log.Error("failed to delete existing product options", zap.Error(err))
			return PutProductId404JSONResponse{
				Message: "update failed",
			}, nil
		}

		// Add new options
		options := make([]models.ProductOption, 0)
		for _, o := range *request.Body.Options {
			options = append(options, models.ProductOption{
				ID:        uuid.New(),
				ProductID: id,
				Name:      o.Name,
				Code:      o.Code,
				Quantity:  o.Quantity,
				Price:     float64(o.Price),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			})
		}
		err = s.sc.ProductOptionRepo.WithTx(tx).CreateMany(ctx, options)
		if err != nil {
			s.sc.Log.Error("failed to create product options", zap.Error(err))
			return PutProductId404JSONResponse{
				Message: "update failed",
			}, nil
		}

	}

	updateData := map[string]any{
		"name":       request.Body.Name,
		"code":       request.Body.Code,
		"img":        request.Body.Img,
		"updated_at": time.Now(),
	}
	err = s.sc.ProductRepo.WithTx(tx).Update(ctx, id, updateData)
	if err != nil {
		s.sc.Log.Error("failed to update product", zap.Error(err))
		return PutProductId404JSONResponse{
			Message: "update failed",
		}, nil
	}

	tx.Commit()

	mess := "update success"
	return PutProductId200JSONResponse{
		Message: &mess,
	}, nil
}
