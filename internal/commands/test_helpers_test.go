package commands

// setTestBaseDir overrides baseDirFn for tests and returns a cleanup function.
func setTestBaseDir(dir string) func() {
	orig := baseDirFn
	baseDirFn = func() string { return dir }
	return func() { baseDirFn = orig }
}
