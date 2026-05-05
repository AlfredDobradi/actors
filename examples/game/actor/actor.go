package actor

import "github.com/alfreddobradi/actors/pkg/system"

func InitFactories(registry *system.Registry) {
	registry.RegisterFactory("TickerActor", tickerActorFactory)
	registry.RegisterFactory("ReceiverActor", receiveActorFactory)
	registry.RegisterFactory("CharacterStore", characterStoreFactory)
	registry.RegisterFactory("AccountActor", accountActorFactory)
}
