package errors

import (
	"errors"
	"io"

	emperror "emperror.dev/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
)

var ErrPodNotReady = errors.New("Pod not ready")
var ErrStsNotReady = errors.New("Sts not ready")

func IsCommonError(err error) bool {
	e := emperror.Cause(err)
	return isEOF(e) ||
		isPodNotReady(e) ||
		isStsNotReady(e) ||
		k8sErrors.IsNotFound(e) ||
		k8sErrors.IsConflict(e)
}

func isPodNotReady(err error) bool {
	return emperror.Is(err, ErrPodNotReady)
}

func isStsNotReady(err error) bool {
	return emperror.Is(err, ErrStsNotReady)
}

func isEOF(err error) bool {
	return err == io.EOF
}
