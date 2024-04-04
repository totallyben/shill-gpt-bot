package api

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	em "github.com/labstack/echo/v4/middleware"
	gl "github.com/labstack/gommon/log"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/storage"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// version  string = "v1"
	// basePath        = "/" + version
	basePath = ""
)

type Api struct {
	port   int
	logger *zap.Logger
	atom   *zap.AtomicLevel
	mongo  *storage.Mongo
}

func NewApi(port int) *Api {
	// see https://pkg.go.dev/go.uber.org/zap#AtomicLevel
	atom := zap.NewAtomicLevel()
	encoderCfg := zap.NewProductionEncoderConfig()
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atom,
	))
	defer logger.Sync()

	api := &Api{
		port:   port,
		logger: logger,
		atom:   &atom,
		mongo:  storage.NewMongo(),
	}

	return api
}

func (a *Api) Serve() {
	// create a new echo instance
	e := echo.New()
	e.Logger.SetLevel(gl.DEBUG)
	e.HideBanner = true

	// initalise the api validator and set as the echo validator
	v := NewValidator()
	e.Validator = v

	e.Use(em.Logger()) // logger em will “wrap” recovery
	// e.Use(em.RequestLoggerWithConfig(em.RequestLoggerConfig{
	// 	LogURI:    true,
	// 	LogStatus: true,
	// 	LogValuesFunc: func(c echo.Context, v em.RequestLoggerValues) error {
	// 		a.logger.Info("request",
	// 			zap.String("URI", v.URI),
	// 			zap.Int("status", v.Status),
	// 		)

	// 		return nil
	// 	},
	// }))

	e.Use(em.Recover()) // as it is enumerated before in the Use calls
	e.Use(em.Gzip())
	// e.Use(em.CORS())

	e.Use(em.CORSWithConfig(em.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
	}))

	// group services by api version
	// g := e.Group(a.BasePath(), a.oidcAuthenticate)
	g := e.Group(a.BasePath())

	// reply service
	ss := newShillService(a)
	ss.LoadRoutes(g)

	// Route / to handler function
	e.GET("/health-check", a.healthCheck)

	// start the server, and log if it fails
	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", a.port)))
}

// Mongo - return bot mongo connection
func (a *Api) Mongo() *storage.Mongo {
	return a.mongo
}

// // Version - get api version
// func (a *Api) Version() string {
// 	return version
// }

// BasePath - get base path for api e.g. /v1
func (a *Api) BasePath() string {
	return basePath
}

func (a *Api) healthCheck(c echo.Context) error {
	return ReturnSuccessMessage(c, "We're alive!")
}

// func (a *Api) oidcAuthenticate(next echo.HandlerFunc) echo.HandlerFunc {
// 	return echo.HandlerFunc(func(c echo.Context) error {
// 		authorised, err := NewOidcMiddleware().Authenticate(c)

// 		if !authorised {
// 			return ReturnNotAuthorised(c, err)
// 		}

// 		return next(c)
// 	})
// }

// Error - to allow ErrorResponse to be used as an error it must use the go error interface
func (er *ErrorResponse) Error() string {
	return fmt.Sprintf("%v", er.Errors)
}

// func returnErrors(code int, c echo.Context, err error) error {

// 	// switch exp := m["exp"].(type) {
// 	et := reflect.TypeOf(err).String()

// 	if et == "*api.ErrorResponse" {
// 		return c.JSON(code, err)
// 	}

// 	er := new(ErrorResponse)
// 	er.Errors = append(er.Errors, err.Error())
// 	return c.JSON(code, er)
// }

func returnMessage(code int, c echo.Context, err error) error {
	r := new(MessageResponse)
	r.Message = err.Error()
	return c.JSON(code, r)
}

// ReturnError - returns a 400 error
func ReturnError(c echo.Context, err error) error {
	return returnMessage(http.StatusBadRequest, c, err)
}

// ReturnFatalError - returns a 500 error
func ReturnFatalError(c echo.Context, err error) error {
	return returnMessage(http.StatusInternalServerError, c, err)
}

// ReturnNotFound - returns a 404 error
func ReturnNotFound(c echo.Context, err error) error {
	return returnMessage(http.StatusNotFound, c, err)
}

// ReturnSuccessMessage - return a 200 success message
func ReturnSuccessMessage(c echo.Context, m string) error {
	r := &MessageResponse{Message: m}
	return c.JSON(http.StatusOK, r)
}

// ReturnSuccessWithData - return a 200 with data
func ReturnSuccessWithData(c echo.Context, i interface{}) error {
	return c.JSON(http.StatusOK, i)
}

// ReturnNotAuthorised - return a 401 unauthorised error
func ReturnNotAuthorised(c echo.Context, err error) error {
	return returnMessage(http.StatusUnauthorized, c, err)
}

// ReturnForbidden - return a 403 forbidden error
func ReturnForbidden(c echo.Context, err error) error {
	return returnMessage(http.StatusForbidden, c, err)
}

// DebugRequest
func DebugRequest(c echo.Context) {
	// Read the request body
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		log.Fatal(err)
	}

	// Log the request body
	println(string(body))
}
