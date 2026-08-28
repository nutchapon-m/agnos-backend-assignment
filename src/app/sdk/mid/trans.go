package mid

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/logger"
)

// GetTran returns the transaction started by the BeginCommitRollback
// middleware. The second return value is false when the handler is not
// running inside a transaction.
func GetTran(c *gin.Context) (sqldb.CommitRollbacker, bool) {
	v, exists := c.Get(trKey)
	if !exists {
		return nil, false
	}

	tx, ok := v.(sqldb.CommitRollbacker)
	return tx, ok
}

func BeginCommitRollback(log *logger.Logger, bgn sqldb.Beginner) gin.HandlerFunc {
	return func(c *gin.Context) {
		hasCommitted := false

		log.Info(c, "BEGIN TRANSACTION")
		tx, err := bgn.Begin()
		if err != nil {
			return
		}

		defer func() {
			if !hasCommitted {
				log.Info(c, "ROLLBACK TRANSACTION")
			}

			if err := tx.Rollback(); err != nil {
				if errors.Is(err, sql.ErrTxDone) {
					return
				}
				log.Info(c, "ROLLBACK TRANSACTION", "ERROR", err)
			}
		}()

		c.Set(trKey, tx)
		c.Next()

		// Handlers report failure by writing an error status through the
		// response package, which does not touch c.Errors. Without this the
		// deferred rollback is preempted by the commit below and a partially
		// applied multi-write handler would be persisted.
		if c.Writer.Status() >= http.StatusBadRequest {
			return
		}

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}

		log.Info(c, "COMMIT TRANSACTION")
		if err := tx.Commit(); err != nil {
			return
		}

		hasCommitted = true
	}
}
