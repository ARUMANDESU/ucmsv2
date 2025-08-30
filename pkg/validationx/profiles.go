package validationx

import (
	"github.com/ARUMANDESU/validation"
	"github.com/ARUMANDESU/validation/is"
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
