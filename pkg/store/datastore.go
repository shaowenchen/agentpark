package store

import (
	"fmt"
	"os"
	"strings"

	"github.com/agentpark/agentpark/pkg/datastore"
	"github.com/agentpark/agentpark/pkg/mysqlstore"
)

// DatastoreFromEnv 根据 AGENTPARK_DATASTORE 构造 datastore.Store。
//
//	AGENTPARK_DATASTORE=memory|mysql（默认 memory）
//	mysql 时需设置 AGENTPARK_MYSQL_DSN，例如：
//	user:pass@tcp(127.0.0.1:3306)/agentpark?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci
func DatastoreFromEnv() (datastore.Store, string, error) {
	kind := strings.ToLower(strings.TrimSpace(os.Getenv("AGENTPARK_DATASTORE")))
	if kind == "" {
		kind = "memory"
	}
	switch kind {
	case "memory":
		return datastore.NewMemory(), kind, nil
	case "mysql":
		dsn := strings.TrimSpace(os.Getenv("AGENTPARK_MYSQL_DSN"))
		if dsn == "" {
			return nil, "", fmt.Errorf("AGENTPARK_DATASTORE=mysql requires AGENTPARK_MYSQL_DSN")
		}
		st, err := mysqlstore.Open(dsn)
		if err != nil {
			return nil, "", err
		}
		return st, kind, nil
	default:
		return nil, "", fmt.Errorf("AGENTPARK_DATASTORE: unsupported %q (memory|mysql)", kind)
	}
}
