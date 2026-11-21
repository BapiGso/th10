package laser

// Inspect visits every linked laser in FUN_0041c330 list order, including
// pending StateDead objects. Unlike Each, it neither prepares nor filters
// sprites; callers must not mutate a laser or run the manager in fn.
func (m *Manager) Inspect(fn func(Laser)) {
	for _, l := range m.list {
		if l != nil {
			fn(l)
		}
	}
}
