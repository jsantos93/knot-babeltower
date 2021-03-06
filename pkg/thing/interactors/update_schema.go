package interactors

import (
	"fmt"

	"github.com/CESARBR/knot-babeltower/pkg/thing/entities"
	"github.com/go-playground/validator"
)

type schemaType struct {
	valueType interface{}
	unit      interface{}
}

type interval struct {
	min int
	max int
}

// rules reference table: https://knot-devel.cesar.org.br/doc/thing/unit-type-value.html
var rules = map[int]schemaType{
	0x0000: schemaType{valueType: 4, unit: 0},              // RAW   => NONE
	0x0001: schemaType{valueType: 1, unit: interval{1, 3}}, // INT   => VOLTAGE
	0x0002: schemaType{valueType: 1, unit: interval{1, 2}}, // INT   => CURRENT
	0x0003: schemaType{valueType: 1, unit: 1},              // INT   => RESISTENCE
	0x0004: schemaType{valueType: 1, unit: interval{1, 3}}, // INT   => POWER
	0x0005: schemaType{valueType: 1, unit: interval{1, 3}}, // INT   => TEMPERATURE
	0x0006: schemaType{valueType: 1, unit: 1},              // INT   => RELATIVE_HUMIDITY
	0x0007: schemaType{valueType: 1, unit: interval{1, 3}}, // INT   => LUMINOSITY
	0x0008: schemaType{valueType: 1, unit: interval{1, 3}}, // INT   => TIME
	0x0009: schemaType{valueType: 1, unit: interval{1, 4}}, // INT   => MASS
	0x000A: schemaType{valueType: 1, unit: interval{1, 3}}, // INT   => PRESSURE
	0x000B: schemaType{valueType: 1, unit: interval{1, 4}}, // INT   => DISTANCE
	0x000C: schemaType{valueType: 2, unit: interval{1, 2}}, // FLOAT => ANGLE
	0x000D: schemaType{valueType: 2, unit: interval{1, 4}}, // FLOAT => VOLUME
	0x000E: schemaType{valueType: 2, unit: interval{1, 3}}, // FLOAT => AREA
	0x000F: schemaType{valueType: 2, unit: 1},              // FLOAT => RAIN
	0x0010: schemaType{valueType: 2, unit: 1},              // FLOAT => DENSITY
	0x0011: schemaType{valueType: 2, unit: 1},              // FLOAT => LATITUDE
	0x0012: schemaType{valueType: 2, unit: 1},              // FLOAT => LONGITUDE
	0x0013: schemaType{valueType: 1, unit: interval{1, 4}}, // INT   => SPEED
	0x0014: schemaType{valueType: 2, unit: interval{1, 6}}, // FLOAT => VOLUMEFLOW
	0x0015: schemaType{valueType: 1, unit: interval{1, 6}}, // INT   => ENERGY
	0xFFF0: schemaType{valueType: 3, unit: 0},              // BOOL  => PRESENCE
	0xFFF1: schemaType{valueType: 3, unit: 0},              // BOOL  => SWITCH
	0xFFF2: schemaType{valueType: 4, unit: 0},              // RAW   => COMMAND
	0xFF10: schemaType{valueType: 1, unit: 0},              // INT   => ANALOG
	0xFFFF: schemaType{valueType: 4, unit: 0},              // RAW   => INVALID
}

// UpdateSchema receive the new sensor schema and update it on the thing's service
func (i *ThingInteractor) UpdateSchema(authorization, thingID string, schemaList []entities.Schema) error {
	if authorization == "" {
		return ErrAuthNotProvided
	}
	if thingID == "" {
		return ErrIDNotProvided
	}
	if schemaList == nil {
		return ErrSchemaNotProvided
	}

	if !i.isValidSchema(schemaList) {
		err := i.notifyClient(thingID, ErrSchemaInvalid)
		return err
	}
	i.logger.Info("updateSchema: schema validated")

	err := i.thingProxy.UpdateSchema(authorization, thingID, schemaList)
	if err != nil {
		sendErr := i.notifyClient(thingID, err)
		return sendErr
	}
	i.logger.Info("updateSchema: schema updated")

	err = i.notifyClient(thingID, err)
	if err != nil {
		// TODO: handle error when publishing message to queue.
		return err
	}
	i.logger.Info("updateSchema: message sent to client")

	err = i.connectorPublisher.SendUpdateSchema(thingID, schemaList)
	if err != nil {
		// TODO: handle error when publishing message to queue.
		return err
	}
	i.logger.Info("updateSchema: message sent to connector")

	return nil
}

func (i *ThingInteractor) isValidSchema(schemaList []entities.Schema) bool {
	validate := validator.New()
	validate.RegisterStructValidation(schemaValidation, entities.Schema{})
	for _, schema := range schemaList {
		err := validate.Struct(schema)
		if err != nil {
			return false
		}
	}

	return true
}

func (i *ThingInteractor) notifyClient(thingID string, err error) error {
	sendErr := i.clientPublisher.SendUpdatedSchema(thingID, err)
	if sendErr != nil {
		if err != nil {
			return fmt.Errorf("error sending response to client: %v: %w", sendErr, err)
		}
		return fmt.Errorf("error sending response to client: %w", sendErr)
	}
	return err
}

func schemaValidation(sl validator.StructLevel) {
	schema := sl.Current().Interface().(entities.Schema)
	typeID := schema.TypeID

	if (typeID < 0 || 15 < typeID) && (typeID < 0xfff0 || 0xfff2 < typeID) && typeID != 0xff10 {
		sl.ReportError(schema, "schema", "Type ID", "typeID", "false")
		return
	}

	if !isValidValueType(schema.TypeID, schema.ValueType) {
		sl.ReportError(schema, "schema", "Value Type", "valueType", "false")
		return
	}

	if !isValidUnit(schema.TypeID, schema.Unit) {
		sl.ReportError(schema, "schema", "Unit", "unit", "false")
	}
}

func isValidValueType(typeID, valueType int) bool {
	t := rules[typeID].valueType
	if t == nil {
		return false
	}

	switch v := t.(type) {
	case int:
		value := v
		if valueType != value {
			return false
		}
	case interval:
		interval := t.(interval)
		if valueType < interval.min || interval.max < valueType {
			return false
		}
	}

	return true
}

func isValidUnit(typeID, unit int) bool {
	u := rules[typeID].unit
	if u == nil {
		return false
	}

	switch v := u.(type) {
	case int:
		value := v
		if unit != value {
			return false
		}
	case interval:
		interval := u.(interval)
		if unit < interval.min || interval.max < unit {
			return false
		}
	}

	return true
}
