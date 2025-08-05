package framework

import (
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/suite"

	registrationapp "github.com/ARUMANDESU/ucms/internal/application/registration"
)

type IntegrationTestSuite struct {
	suite.Suite

	app         *Application
	httpHandler chi.Router

	HTTP    *HTTPHelper
	DB      *DBHelper
	Event   *EventHelper
	Builder *BuilderFactory
}

type Application struct {
	registration *registrationapp.App
}
