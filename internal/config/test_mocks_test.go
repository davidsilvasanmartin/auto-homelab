package config

// mockPrompter is a mock implementation of Prompter for testing
type mockPrompter struct {
	promptFunc func(message string) (string, error)
	infoFunc   func(message string)
}

func (m *mockPrompter) Prompt(message string) (string, error) {
	if m.promptFunc != nil {
		return m.promptFunc(message)
	}
	return "", nil
}
func (m *mockPrompter) Info(message string) {
	if m.infoFunc != nil {
		m.infoFunc(message)
	}
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
	ensureDirExists      func(path string) error
	getAbsPath           func(path string) (string, error)
	getwd                func() (string, error)
	writeFile            func(path string, data []byte) error
}

func (m *mockFiles) CreateDirIfNotExists(path string) error {
	if m.createDirIfNotExists != nil {
		return m.createDirIfNotExists(path)
	}
	return nil
}
func (m *mockFiles) EnsureFilesInWD(filenames ...string) error { return nil }
func (m *mockFiles) EnsureDirExists(path string) error {
	if m.ensureDirExists != nil {
		return m.ensureDirExists(path)
	}
	return nil
}
func (m *mockFiles) EmptyDir(path string) error {
	return nil
}
func (m *mockFiles) CopyDir(srcPath string, dstPath string) error {
	return nil
}
func (m *mockFiles) Getwd() (dir string, err error) {
	if m.getwd != nil {
		return m.getwd()
	}
	return "", nil
}
func (m *mockFiles) WriteFile(path string, data []byte) error {
	if m.writeFile != nil {
		return m.writeFile(path, data)
	}
	return nil
}
func (m *mockFiles) GetAbsPath(path string) (string, error) {
	if m.getAbsPath != nil {
		return m.getAbsPath(path)
	}
	return "", nil
}

type mockStrategyRegistry struct {
	getFunc func(varType string) (AcquireStrategy, error)
}

func (m *mockStrategyRegistry) Get(varType string) (AcquireStrategy, error) {
	if m.getFunc != nil {
		return m.getFunc(varType)
	}
	return nil, nil
}
func (m *mockStrategyRegistry) Register(typeName string, strategy AcquireStrategy) {}

// mockStrategy is a mock implementation of AcquireStrategy for testing
type mockStrategy struct {
	acquireFunc func(varName string, defaultSpec *string) (string, error)
}

func (m *mockStrategy) Acquire(varName string, defaultSpec *string) (string, error) {
	if m.acquireFunc != nil {
		return m.acquireFunc(varName, defaultSpec)
	}
	return "mock-value", nil
}

type mockTextFormatter struct {
	formatDotenvKeyValue func(key string, value string) (string, error)
}

func (m *mockTextFormatter) WrapLines(text string, width uint) []string {
	return []string{text}
}
func (m *mockTextFormatter) FormatDotenvKeyValue(key string, value string) (string, error) {
	if m.formatDotenvKeyValue != nil {
		return m.formatDotenvKeyValue(key, value)
	}
	return key + "=" + value, nil
}
func (m *mockTextFormatter) QuoteForPOSIXShell(text string) string {
	return text
}
