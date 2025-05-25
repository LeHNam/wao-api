package purchase_order

import (
	"context"
	"fmt"
	svCtx "github.com/LeHNam/wao-api/context"
	"github.com/LeHNam/wao-api/helpers/utils"
	"github.com/LeHNam/wao-api/services/database"
	"github.com/LeHNam/wao-api/services/websocket"
	"math/rand"
	"time"

	"github.com/LeHNam/wao-api/models"
	"github.com/google/uuid"
)

type PurchaseOrderServer struct {
	sc        *svCtx.ServiceContext
	wsService *websocket.WebSocketService
}

func NewPurchaseOrderServer(sc *svCtx.ServiceContext, wsService *websocket.WebSocketService) *PurchaseOrderServer {
	return &PurchaseOrderServer{
		sc:        sc,
		wsService: wsService,
	}
}
func GenerateOrderNumber() string {
	rand.Seed(time.Now().UnixNano())
	randomPart := rand.Intn(10000)                       // Generate a random number between 0 and 9999
	timestampPart := time.Now().Format("20060102150405") // Format: YYYYMMDDHHMMSS
	return fmt.Sprintf("PO-%s-%04d", timestampPart, randomPart)
}
func (s *PurchaseOrderServer) PostPurchaseOrder(ctx context.Context, request PostPurchaseOrderRequestObject) (PostPurchaseOrderResponseObject, error) {
	userCtx := utils.GetUserFromContext(ctx)

	productIds := make([]uuid.UUID, 0, len(request.Body.Items))
	for _, item := range request.Body.Items {
		productIds = append(productIds, item.ProductId)
	}

	cond := map[string]any{
		"id IN": productIds,
	}
	products, err := s.sc.ProductRepo.FindWithJoinAndPreload(ctx, cond, []string{}, 0, 0, nil, []string{}, []database.PreloadData{
		{Field: "Options"},
	})

	if err != nil {
		return PostPurchaseOrder400JSONResponse{
			Message: utils.Stp("Failed to fetch products"),
		}, nil
	}

	mappedProducts := make(map[string]models.Product)
	mappedProductOptions := make(map[string]models.ProductOption)
	for _, p := range products {
		mappedProducts[p.ID.String()] = p
		for _, option := range p.Options {
			mappedProductOptions[p.ID.String()+"_"+option.ID.String()] = option
		}
	}

	purchaseOrderId := uuid.New()
	totalAmount := 0.0
	// Create purchase order items
	items := make([]models.PurchaseOrderItem, 0, len(request.Body.Items))
	for _, item := range request.Body.Items {
		product, exists := mappedProducts[item.ProductId.String()]
		if !exists {
			return PostPurchaseOrder400JSONResponse{
				Message: utils.Stp("Product not found: " + item.ProductId.String()),
			}, nil
		}

		option, exists := mappedProductOptions[item.ProductId.String()+"_"+item.ProductOptionId.String()]
		if !exists {
			return PostPurchaseOrder400JSONResponse{
				Message: utils.Stp("Product option not found: " + item.ProductOptionId.String()),
			}, nil
		}

		if option.Quantity <= 0 || option.Price <= 0 {
			return PostPurchaseOrder400JSONResponse{
				Message: utils.Stp("Product option is not available: " + item.ProductOptionId.String()),
			}, nil
		}

		// Calculate total price based on quantity and unit price
		totalPrice := float64(item.Quantity) * option.Price
		if totalPrice <= 0 {
			return PostPurchaseOrder400JSONResponse{
				Message: utils.Stp("Total price must be greater than zero"),
			}, nil
		}
		totalAmount += totalPrice
		items = append(items, models.PurchaseOrderItem{
			ID:                uuid.New(),
			ProductID:         item.ProductId,
			ProductName:       product.Name,
			ProductOptionID:   item.ProductOptionId,
			ProductOptionName: option.Name,
			UnitPrice:         option.Price,
			TotalPrice:        totalPrice,
			Quantity:          item.Quantity,
			Currency:          item.Currency,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
			PurchaseOrderID:   purchaseOrderId,
		})
	}

	// Create purchase order
	purchaseOrder := &models.PurchaseOrder{
		ID:          purchaseOrderId,
		OrderNumber: GenerateOrderNumber(),
		Status:      "DRAFT",
		TotalAmount: totalAmount,
		OrderDate:   time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   userCtx.ID,
	}

	tx := s.sc.DB.Begin().WithContext(ctx)
	if tx.Error != nil {
		s.sc.Log.Error(tx.Error.Error())
		return PostPurchaseOrder400JSONResponse{
			Message: utils.Stp("Create DB transaction failed"),
		}, nil
	}
	defer tx.Rollback()

	err = s.sc.PurchaseOrderRepo.WithTx(tx).Create(ctx, purchaseOrder)
	if err != nil {
		return PostPurchaseOrder400JSONResponse{
			Message: utils.Stp("Failed to create purchase order"),
		}, nil
	}

	err = s.sc.PurchaseOrderItemRepo.WithTx(tx).CreateMany(ctx, items)
	if err != nil {
		return PostPurchaseOrder400JSONResponse{
			Message: utils.Stp("Failed to create purchase order items"),
		}, nil
	}

	tx.Commit()
	return PostPurchaseOrder200JSONResponse{
		Id: &purchaseOrder.ID,
	}, nil
}

func (s *PurchaseOrderServer) GetPurchaseOrder(ctx context.Context, request GetPurchaseOrderRequestObject) (GetPurchaseOrderResponseObject, error) {
	userCtx := utils.GetUserFromContext(ctx)

	cond := map[string]any{}

	if userCtx.Role == "buyer" {
		cond["created_by"] = userCtx.ID
	}
	orders, err := s.sc.PurchaseOrderRepo.Find(ctx, cond, []string{}, 0, 0, nil)
	if err != nil {
		return GetPurchaseOrder500JSONResponse{
			Message: utils.Stp("Failed to fetch purchase orders"),
		}, nil
	}

	response := make([]PurchaseOrder, 0, len(orders))
	for _, order := range orders {
		response = append(response, PurchaseOrder{
			Id:          order.ID,
			Status:      order.Status,
			OrderDate:   order.OrderDate,
			TotalAmount: float32(order.TotalAmount),
			Currency:    order.Currency,
			OrderNumber: order.OrderNumber,
		})
	}

	return GetPurchaseOrder200JSONResponse(response), nil
}

func (s *PurchaseOrderServer) GetPurchaseOrderId(ctx context.Context, request GetPurchaseOrderIdRequestObject) (GetPurchaseOrderIdResponseObject, error) {

	order, err := s.sc.PurchaseOrderRepo.First(ctx, request.Id)
	if err != nil {
		return GetPurchaseOrderId404JSONResponse{
			Message: utils.Stp("Purchase order not found"),
		}, nil
	}

	items, err := s.sc.PurchaseOrderItemRepo.Find(ctx, map[string]any{"purchase_order_id": order.ID}, []string{}, 0, 0, nil)
	if err != nil {
		return GetPurchaseOrderId500JSONResponse{
			Message: utils.Stp("Failed to fetch purchase order items"),
		}, nil
	}
	responseItems := make([]PurchaseOrderItem, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, PurchaseOrderItem{
			Id:                item.ID,
			ProductId:         item.ProductID,
			ProductName:       item.ProductName,
			ProductOptionId:   item.ProductOptionID,
			ProductOptionName: item.ProductOptionName,
			UnitPrice:         float32(item.UnitPrice),
			TotalPrice:        float32(item.TotalPrice),
			Currency:          item.Currency,
			Quantity:          item.Quantity,
		})
	}
	return GetPurchaseOrderId200JSONResponse{
		Id:          order.ID,
		Status:      order.Status,
		OrderDate:   order.OrderDate,
		TotalAmount: float32(order.TotalAmount),
		Currency:    order.Currency,
		OrderNumber: order.OrderNumber,
		Items:       &responseItems,
	}, nil
}

func (s *PurchaseOrderServer) PatchPurchaseOrderIdStatus(ctx context.Context, request PatchPurchaseOrderIdStatusRequestObject) (PatchPurchaseOrderIdStatusResponseObject, error) {

	updateData := map[string]any{
		"status":     request.Body.Status,
		"updated_at": time.Now(),
	}

	order, err := s.sc.PurchaseOrderRepo.First(ctx, request.Id)
	if err != nil {
		return PatchPurchaseOrderIdStatus400JSONResponse{
			Message: utils.Stp("Purchase order not found"),
		}, nil
	}

	err = s.sc.PurchaseOrderRepo.Update(ctx, request.Id, updateData)
	if err != nil {
		return PatchPurchaseOrderIdStatus400JSONResponse{
			Message: utils.Stp("Failed to update status"),
		}, nil
	}
	order.Status = request.Body.Status
	message := map[string]any{
		"event": "order_updated",
		"data":  order,
	}
	_ = s.wsService.Broadcast(message)

	success := true
	return PatchPurchaseOrderIdStatus200JSONResponse{
		Success: &success,
	}, nil
}
