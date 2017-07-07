package main

import "github.com/Pallinder/go-randomdata"
import "time"

type Field interface {
	Type() string
	GenerateValue() interface{}
}

type StringField struct {
	length int
}

func (field StringField) Type() string {
	return "string"
}

func (field StringField) GenerateValue() interface{} {
	return randomdata.RandStringRunes(field.length)
}

type IntegerField struct {
	min int
	max int
}

func (field IntegerField) Type() string {
	return "integer"
}

func (field IntegerField) GenerateValue() interface{} {
	return randomdata.Number(field.min, field.max)
}

type FloatField struct {
}

type DateField struct {
	min string
	max string
}

func (field DateField) Type() string {
	return "date"
}

func (field DateField) ValidBounds() bool {
	// NOTE: The time package parses/formats dates relative to the following date:
	// * Mon Jan 2 15:04:05 -0700 MST 2006
	//
	// Why????? I have no idea
	// so, the constant timeFormat cannot deviate from that specific date.
	// That said, the specific time is only needed if the date in question
	// has a time.
	// see: https://golang.org/pkg/time/#Parse
	const timeFormat = "2006-01-02"
	min, _ := time.Parse(timeFormat, field.min)
	max, _ := time.Parse(timeFormat, field.max)
	if min.Before(max) {
		return true
	} else {
		return false
	}
}

func (field DateField) GenerateValue() interface{} {
	return randomdata.FullDateInRange(field.min, field.max)
}