package handler

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz/policy"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/handler"
	"github.com/skygeario/skygear-server/pkg/core/inject"
	"github.com/skygeario/skygear-server/pkg/core/server"
	recordGear "github.com/skygeario/skygear-server/pkg/record"
	"github.com/skygeario/skygear-server/pkg/record/dependency/record"
)

func AttachFieldAccessUpdateHandler(
	server *server.Server,
	recordDependency recordGear.DependencyMap,
) *server.Server {
	server.Handle("/schema/field_access/update", &FieldAccessUpdateHandlerFactory{
		recordDependency,
	}).Methods("POST")
	return server
}

type FieldAccessUpdateHandlerFactory struct {
	Dependency recordGear.DependencyMap
}

func (f FieldAccessUpdateHandlerFactory) NewHandler(request *http.Request) http.Handler {
	h := &FieldAccessUpdateHandler{}
	inject.DefaultInject(h, f.Dependency, request)
	return handler.APIHandlerToHandler(h, h.TxContext)
}

func (f FieldAccessUpdateHandlerFactory) ProvideAuthzPolicy() authz.Policy {
	return policy.AllOf(
		authz.PolicyFunc(policy.RequireMasterKey),
	)
}

type FieldAccessUpdateRequestPayload struct {
}

func (s FieldAccessUpdateRequestPayload) Validate() error {
	return nil
}

/*
FieldAccessUpdateHandler fetches the entire Field ACL settings.
curl -X POST -H "Content-Type: application/json" \
  -d @- http://localhost:3000/schema/field_access/update <<EOF
{
	"access": [
		{
			"record_type":"note",
			"record_field":"content",
			"user_role":"_user_id:johndoe",
			"writable":false,
			"readable":true,
			"comparable":false,
			"discoverable":false
		}
	]
}
EOF
*/
type FieldAccessUpdateHandler struct {
	TxContext   db.TxContext  `dependency:"TxContext"`
	RecordStore record.Store  `dependency:"RecordStore"`
	Logger      *logrus.Entry `dependency:"HandlerLogger"`
}

func (h FieldAccessUpdateHandler) WithTx() bool {
	return true
}

func (h FieldAccessUpdateHandler) DecodeRequest(request *http.Request) (handler.RequestPayload, error) {
	payload := FieldAccessUpdateRequestPayload{}
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		return nil, err
	}

	return payload, nil
}

func (h FieldAccessUpdateHandler) Handle(req interface{}) (resp interface{}, err error) {
	return
}
