package room

import "github.com/fairwix/resource-booking-service/internal/model"

func (r Room) Status() model.BookingStatus {
	return r.status
}
