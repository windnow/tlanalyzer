package monitor

func (m *Monitor) getFolders() []Log {
	m.mutex.Lock()
	result := append(m.folders, m.cfg_folders...)
	m.mutex.Unlock()
	return result
}
func (m *Monitor) setFolders(cfg_folders []Log) {
	m.mutex.Lock()
	m.cfg_folders = cfg_folders
	m.mutex.Unlock()
}
