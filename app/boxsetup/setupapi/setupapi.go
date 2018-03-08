package setupapi

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/NAVCoin/navpi-go/app/api"
	"github.com/NAVCoin/navpi-go/app/conf"
	"github.com/NAVCoin/navpi-go/app/middleware"
	"github.com/gorilla/mux"
	"github.com/muesli/crunchy"
)

type UIProtection struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// InitSetupHandlers sets the api
func InitSetupHandlers(r *mux.Router, prefix string) {

	var nameSpace string = "setup"

	r.Handle(fmt.Sprintf("/%s/%s/v1/setrange", prefix, nameSpace), middleware.Adapt(rangeSetHandler()))

	// Protect UI with username and password
	r.Handle(fmt.Sprintf("/%s/%s/v1/protectui", prefix, nameSpace), middleware.Adapt(protectUIHandler())).Methods("POST")

}

// rangeSetHandler takes the users ip address and saves it to the config as a range
func protectUIHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var uiProtection UIProtection
		apiResp := api.Response{}

		//Get the json from the post data
		err := json.NewDecoder(r.Body).Decode(&uiProtection)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			returnErr := api.AppRespErrors.ServerError
			returnErr.ErrorMessage = fmt.Sprintf("Server error: %v", err)
			apiResp.Errors = append(apiResp.Errors, returnErr)
			apiResp.Send(w)

			return
		}

		// Check we have a username and password
		if uiProtection.Username == "" || uiProtection.Password == "" {

			w.WriteHeader(http.StatusBadRequest)

			returnErr := api.AppRespErrors.SetupAPIProtectUI
			apiResp.Errors = append(apiResp.Errors, returnErr)
			apiResp.Send(w)

			return

		}

		// Check the password strength
		validator := crunchy.NewValidator()
		err = validator.Check(uiProtection.Password)

		if err != nil {

			w.WriteHeader(http.StatusBadRequest)

			returnErr := api.AppRespErrors.InvalidPasswordStrength
			returnErr.ErrorMessage = fmt.Sprintf("The password is considered unsafe: %v", err)

			apiResp.Errors = append(apiResp.Errors, returnErr)
			apiResp.Send(w)

			return

		}

		// has the details for later
		hashedDetails, err := api.HashDetails(uiProtection.Username, uiProtection.Password)

		// if there was an error hasing the details then error
		if err != nil {

			w.WriteHeader(http.StatusInternalServerError)

			returnErr := api.AppRespErrors.ServerError
			returnErr.ErrorMessage = fmt.Sprintf("The password is considered unsafe: %v", err)
			apiResp.Errors = append(apiResp.Errors, returnErr)
			apiResp.Send(w)

			return

		}

		// Everything is good store the hash
		conf.AppConf.UIPassword = hashedDetails
		conf.SaveAppConfig()
		apiResp.Send(w)

	})
}

// rangeSetHandler takes the users ip address and saves it to the config as a range
func rangeSetHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		host, _, err := net.SplitHostPort(r.RemoteAddr)
		apiResp := api.Response{}

		// If there is no host found we need to error out
		if err != nil || host == "" {

			w.WriteHeader(http.StatusInternalServerError)

			returnErr := api.AppRespErrors.SetupAPINoHost
			apiResp.Errors = append(apiResp.Errors, returnErr)
			apiResp.Send(w)

			return

		}

		// Note "::1"  is the ipV6 version of localhost
		// Check to see we are not using "localhost" - we need an ip
		if host == "::1" {

			w.WriteHeader(http.StatusBadRequest)

			returnErr := api.AppRespErrors.SetupAPIUsingLocalHost
			apiResp.Errors = append(apiResp.Errors, returnErr)

			apiResp.Send(w)
			return

		}

		// we made it here so we are good - so set the config and save to the file

		// separate and make the range wildcard
		strSplit := strings.Split(host, ".")
		strSplit[len(strSplit)-1] = "*"
		strSplit[len(strSplit)-2] = "*"
		host = strings.Join(strSplit, ".")

		conf.AppConf.AllowedIps = append(conf.AppConf.AllowedIps, host)
		conf.SaveAppConfig()

		//Set the rep data
		apiResp.Data = host
		apiResp.Send(w)

	})
}