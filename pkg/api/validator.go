package api

import (
	"log"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"

	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

// Validator - struct to hold our custom echo validator
type Validator struct {
	Validator *validator.Validate
	trans     ut.Translator
}

// Validate - the validate method calls go-playground/validator Validator.Struct()
func (v *Validator) Validate(i interface{}) error {
	err := v.Validator.Struct(i)

	if err == nil {
		return err
	}

	er := new(ErrorResponse)
	for _, e := range err.(validator.ValidationErrors) {
		er.Errors = append(er.Errors, e.Translate(v.trans))
	}

	return er
}

// NewValidator - creates a new Validator instance and initiliases translations
func NewValidator() *Validator {
	translator := en.New()

	uni := ut.New(translator, translator)
	trans, _ := uni.GetTranslator("en")

	v := validator.New()

	if err := en_translations.RegisterDefaultTranslations(v, trans); err != nil {
		log.Fatal(err)
	}

	_ = v.RegisterTranslation("required", trans, func(ut ut.Translator) error {
		return ut.Add("required", "{0} is a required field", true) // see universal-translator for details
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("required", fe.Field())
		return t
	})

	_ = v.RegisterTranslation("email", trans, func(ut ut.Translator) error {
		return ut.Add("email", "{0} must be a valid email", true) // see universal-translator for details
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("email", fe.Field())
		return t
	})

	return &Validator{Validator: v, trans: trans}
}
