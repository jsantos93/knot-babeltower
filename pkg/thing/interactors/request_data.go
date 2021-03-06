package interactors

import (
	"fmt"

	"github.com/CESARBR/knot-babeltower/pkg/thing/entities"
)

// RequestData executes the use case operations to request data from the thing
func (i *ThingInteractor) RequestData(authorization, thingID string, sensorIds []int) error {
	if authorization == "" {
		return ErrAuthNotProvided
	}
	if thingID == "" {
		return ErrIDNotProvided
	}
	if sensorIds == nil {
		return ErrSensorsNotProvided
	}

	thing, err := i.thingProxy.Get(authorization, thingID)
	if err != nil {
		i.logger.Error(err)
		return err
	}

	if thing.Schema == nil {
		i.logger.Error(fmt.Errorf("thing %s has no schema yet", thing.ID))
		return err
	}

	err = validateSensors(sensorIds, thing.Schema)
	if err != nil {
		i.logger.Error(err)
		return err
	}

	err = i.clientPublisher.SendRequestData(thingID, sensorIds)
	if err != nil {
		i.logger.Error(err)
		return err
	}

	i.logger.Info("data request command successfully sent")
	return nil
}

// validateSensors validates a slice of sensor ids against the thing's registered schema
// that represents the sensors and actuators associated to it.
func validateSensors(sensorIds []int, schema []entities.Schema) error {
	for _, id := range sensorIds {
		if !sensorExists(schema, id) {
			return ErrSensorInvalid
		}
	}

	return nil
}

func sensorExists(schema []entities.Schema, id int) bool {
	for _, s := range schema {
		if s.SensorID == id {
			return true
		}
	}

	return false
}
