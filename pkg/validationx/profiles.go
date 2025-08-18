package validationx

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

var (
	EmailRules = []validation.Rule{
		validation.Required,
		is.Email,
		validation.Length(5, 255),
	}

	NameRules = []validation.Rule{
		validation.Required,
		validation.Length(1, 150),
		IsPersonName,
	}

	PasswordRules = []validation.Rule{
		validation.Required,
		validation.Length(8, 128),
		PasswordFormat,
	}
)
