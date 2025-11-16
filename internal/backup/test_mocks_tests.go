package backup

type mockFilesHandler struct {
	createDirIfNotExists func(path string) error
	ensureDirExists      func(path string) error
	copyDir              func(srcPath string, dstPath string) error
}

func (m *mockFilesHandler) CreateDirIfNotExists(path string) error {
	if m.createDirIfNotExists != nil {
		return m.createDirIfNotExists(path)
	}
	return nil
}
func (m *mockFilesHandler) EnsureFilesInWD(filenames ...string) error { return nil }
func (m *mockFilesHandler) EnsureDirExists(path string) error {
	if m.ensureDirExists != nil {
		return m.ensureDirExists(path)
	}
	return nil
}
func (m *mockFilesHandler) EmptyDir(path string) error { return nil }
func (m *mockFilesHandler) CopyDir(srcPath string, dstPath string) error {
	if m.copyDir != nil {
		return m.copyDir(srcPath, dstPath)
	}
	return nil
}
func (m *mockFilesHandler) Getwd() (dir string, err error)           { return "", nil }
func (m *mockFilesHandler) WriteFile(path string, data []byte) error { return nil }
func (m *mockFilesHandler) GetAbsPath(path string) (string, error)   { return "", nil }
