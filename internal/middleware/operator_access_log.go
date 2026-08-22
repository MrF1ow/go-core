package middleware

import (
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/MrF1ow/go-core/internal/operator"
)

// OperatorAccessLogger receives a permission decision after policy filtering.
type OperatorAccessLogger func(operator.AccessRecord)

var (
	operatorAccessLoggerMu sync.RWMutex
	operatorAccessLogger   OperatorAccessLogger
)

// SetOperatorAccessLogger installs the best-effort insert hook. Nil skips logging.
func SetOperatorAccessLogger(fn OperatorAccessLogger) {
	operatorAccessLoggerMu.Lock()
	operatorAccessLogger = fn
	operatorAccessLoggerMu.Unlock()
}

func maybeLogOperatorAccess(c *gin.Context, p *operator.Principal, resource, action, decision string, status int) {
	rec := operator.AccessRecord{
		Kind:      p.Kind,
		KeyID:     p.KeyID,
		AccountID: p.AccountID,
		RoleName:  p.RoleName,
		Method:    c.Request.Method,
		Path:      c.Request.URL.Path,
		Decision:  decision,
		Resource:  resource,
		Action:    action,
		Status:    status,
	}
	if !operator.ShouldLogAccess(rec) {
		return
	}
	operatorAccessLoggerMu.RLock()
	fn := operatorAccessLogger
	operatorAccessLoggerMu.RUnlock()
	if fn == nil {
		return
	}
	fn(rec)
}
