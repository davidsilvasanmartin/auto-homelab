package config

// mockPrompter is a mock implementation of Prompter for testing
type mockPrompter struct {
	promptFunc func(message string) (string, error)
	infoFunc   func(message string) error
}

func (m *mockPrompter) Prompt(message string) (string, error) {
	if m.promptFunc != nil {
		return m.promptFunc(message)
	}
	return "", nil
}
func (m *mockPrompter) Info(message string) error {
	if m.infoFunc != nil {
		return m.infoFunc(message)
	}
	return nil
}

type mockEnv struct {
	getEnvFunc func(varName string) (string, bool)
}

// GetEnv assumes by default that the variable does NOT exist
func (m *mockEnv) GetEnv(varName string) (string, bool) {
	if m.getEnvFunc != nil {
		return m.getEnvFunc(varName)
	}
	return "", false
}
func (m *mockEnv) GetRequiredEnv(varName string) (string, error) { return "", nil }

type mockFiles struct {
	createDirIfNotExists func(path string) error
	requireDir           func(path string) error
	getAbsPath           func(path string) (string, error)
}

func (m *mockFiles) CreateDirIfNotExists(path string) error {
	if m.createDirIfNotExists != nil {
		return m.createDirIfNotExists(path)
	}
	return nil
}
func (m *mockFiles) RequireFilesInWd(filenames ...string) error { return nil }
func (m *mockFiles) RequireDir(path string) error {
	if m.requireDir != nil {
		return m.requireDir(path)
	}
	return nil
}
func (m *mockFiles) EmptyDir(path string) error {
	return nil
}
func (m *mockFiles) CopyDir(srcPath string, dstPath string) error {
	return nil
}
func (m *mockFiles) Getwd() (dir string, err error)           { return "", nil }
func (m *mockFiles) WriteFile(path string, data []byte) error { return nil }
func (m *mockFiles) GetAbsPath(path string) (string, error) {
	if m.getAbsPath != nil {
		return m.getAbsPath(path)
	}
	return "", nil
}
