package utils

import "time"

const timeFormateLayout = "2006-01-02 15:04:05"

type Time time.Time

func (t Time) MarshalJSON() ([]byte, error) {
	return []byte(time.Time(t).Format(timeFormateLayout)), nil
}

func (t *Time) UnmarshalJSON(bytes []byte) error {
	now, err := time.ParseInLocation(timeFormateLayout, string(bytes), time.Local)
	if err != nil {
		return err
	}
	*t = Time(now)
	return nil
}
