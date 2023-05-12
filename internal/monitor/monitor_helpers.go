package monitor

func (m *Monitor) getFolders() []Log {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	result := append(m.folders, m.cfg_folders...)
	return result
}
func (m *Monitor) setFolders(cfg_folders []Log) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.cfg_folders = cfg_folders
}
