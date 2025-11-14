package rewardtypes

// Metadata types for different reward types

type DiscountMetadata struct {
	DiscountType string  `json:"discount_type"` // "amount" or "percent"
	Amount       float64 `json:"amount"`
	MinBasket    float64 `json:"min_basket,omitempty"`
	ValidDays    int     `json:"valid_days"`
}

type ExternalVoucherMetadata struct {
	SupplierID string `json:"supplier_id"`
	ProductID  string `json:"product_id"`
}

type PhysicalItemMetadata struct {
	ItemName         string   `json:"item_name"`
	PickupLocations  []string `json:"pickup_locations,omitempty"`
	CollectionPeriod int      `json:"collection_period"` // days
}

type WebhookMetadata struct {
	WebhookURL string            `json:"webhook_url"`
	Secret     string            `json:"secret"`
	Headers    map[string]string `json:"headers,omitempty"`
}

type PointsMetadata struct {
	PointsAmount int    `json:"points_amount"`
	PointsType   string `json:"points_type,omitempty"`
}
