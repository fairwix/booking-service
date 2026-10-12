package desk

import (
	"github.com/fairwix/resource-booking-service/internal/model"
)

func (d Desk) Status() model.BookingStatus {
	return d.status
}
