package nats

import (
	"github.com/File-Sharing-BondBridg/File-Service/internal/api/handlers"
	"github.com/File-Sharing-BondBridg/File-Service/internal/api/handlers/user"
	"github.com/nats-io/nats.go"
)

func Routes() map[string]nats.MsgHandler {
	return map[string]nats.MsgHandler{

		// User events
		"users.deleted": user.HandleUserDeleted,

		// File events
		"files.uploaded": handlers.HandleFileUploaded,
		//"files.deleted":  handlers.HandleFileDeleted,
	}
}
