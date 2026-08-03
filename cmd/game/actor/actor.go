package actor

import (
	"github.com/alfreddobradi/actors/pkg/system"
)

func InitFactories(registry *system.Registry) {
	registry.RegisterFactory("TickerActor", tickerActorFactory)
	registry.RegisterFactory("AccountActor", accountActorFactory)
	registry.RegisterFactory("KeeperActor", keeperActorFactory)
}
