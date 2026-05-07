package actor

import (
	"time"

	"github.com/alfreddobradi/actors/pkg/system"
)

func InitFactories(registry *system.Registry) {
	registry.RegisterFactory("TickerActor", tickerActorFactory)
	registry.RegisterFactory("CharacterStore", characterStoreFactory)
	registry.RegisterFactory("AccountActor", accountActorFactory)
}

type Snapshot struct {
	Timestamp time.Time
	Data      []byte
}
