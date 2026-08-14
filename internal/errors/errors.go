package errors

import (
	"io"

	emperror "emperror.dev/errors"
	"github.com/emqx/emqx-operator/internal/emqx/api"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
)

func IsCommonError(err error) bool {
	e := emperror.Cause(err)
	return e == io.EOF ||
		api.IsUnavailable(e) ||
		api.IsConnectionClosed(e) ||
		k8sErrors.IsNotFound(e) ||
		k8sErrors.IsConflict(e)
}

func IsTransientAPIError(err error) bool {
	e := emperror.Cause(err)
	return e == io.EOF ||
		api.IsUnavailable(e) ||
		api.IsConnectionClosed(e)
}
