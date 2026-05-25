package billing

type service struct{}

func New() BillingService {
	return &service{}
}
