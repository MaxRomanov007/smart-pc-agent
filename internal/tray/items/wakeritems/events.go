package wakeritems

// Events позволяет внешним компонентам (например, HTTP-хендлерам) уведомлять
// tray item о событиях, которые должны отразиться на меню.
type Events struct {
	onAuthorized chan struct{}
}

func NewEvents() *Events {
	return &Events{
		onAuthorized: make(chan struct{}, 1),
	}
}

// NotifyAuthorized сигнализирует, что waker был успешно авторизован.
// Вызов не блокирует — если никто не слушает, сигнал просто дропается.
func (e *Events) NotifyAuthorized() {
	select {
	case e.onAuthorized <- struct{}{}:
	default:
	}
}
