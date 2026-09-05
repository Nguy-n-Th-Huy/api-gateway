package controller

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file closes the test gap that let both setup-script criticals through
// review: the table tests in setup_script_test.go only assert string
// containment against generated script *source*, never actually running the
// hand-rolled JSON/TOML/YAML editors. These tests execute each of the five
// merges — both the Node.js payload and the new native-PowerShell payload —
// against real temporary files, with HOME/USERPROFILE pointed at an isolated
// t.TempDir() so nothing ever touches the real user profile.
//
// A required interpreter that is missing from this machine's PATH skips the
// affected subtests (never fails them).

// ---- interpreter execution helpers -----------------------------------------

// isolatedHomeEnv returns a child-process environment with HOME and
// USERPROFILE pointed at homeDir, so os.homedir() (Node) and $env:USERPROFILE
// (PowerShell) both resolve inside the isolated temp directory rather than
// the real user profile. Every other inherited variable (notably PATH) is
// kept so the child process can still find node/powershell.
func isolatedHomeEnv(homeDir string) []string {
	env := make([]string, 0, len(os.Environ())+2)
	for _, e := range os.Environ() {
		upper := strings.ToUpper(e)
		if strings.HasPrefix(upper, "HOME=") || strings.HasPrefix(upper, "USERPROFILE=") {
			continue
		}
		env = append(env, e)
	}
	return append(env, "HOME="+homeDir, "USERPROFILE="+homeDir)
}

func lookupPowerShellPath() (string, error) {
	if p, err := exec.LookPath("powershell.exe"); err == nil {
		return p, nil
	}
	return exec.LookPath("pwsh")
}

// runNodeMerge writes jsPayload to a temp .js file and executes it with
// node, with HOME/USERPROFILE isolated to homeDir.
func runNodeMerge(t *testing.T, jsPayload string, homeDir string) ([]byte, error) {
	t.Helper()
	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not found on PATH; skipping")
	}
	scriptPath := filepath.Join(t.TempDir(), "merge.js")
	require.NoError(t, os.WriteFile(scriptPath, []byte(jsPayload), 0o644))

	cmd := exec.Command(nodePath, scriptPath)
	cmd.Env = isolatedHomeEnv(homeDir)
	return cmd.CombinedOutput()
}

// runPowerShellMerge writes psPayload to a temp .ps1 file and executes it
// with powershell.exe (falling back to pwsh), with HOME/USERPROFILE
// isolated to homeDir.
func runPowerShellMerge(t *testing.T, psPayload string, homeDir string) ([]byte, error) {
	t.Helper()
	psPath, err := lookupPowerShellPath()
	if err != nil {
		t.Skip("powershell not found on PATH; skipping")
	}
	scriptPath := filepath.Join(t.TempDir(), "merge.ps1")
	require.NoError(t, os.WriteFile(scriptPath, []byte(psPayload), 0o644))

	cmd := exec.Command(psPath, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
	cmd.Env = isolatedHomeEnv(homeDir)
	return cmd.CombinedOutput()
}

func writeConfigFile(t *testing.T, home string, relPath []string, content string) {
	t.Helper()
	full := filepath.Join(append([]string{home}, relPath...)...)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
}

func readConfigFile(t *testing.T, home string, relPath []string) string {
	t.Helper()
	full := filepath.Join(append([]string{home}, relPath...)...)
	data, err := os.ReadFile(full)
	require.NoErrorf(t, err, "expected config file to exist at %s", full)
	return string(data)
}

// ---- per-application merge fixtures -----------------------------------------

const roundtripBaseURL = "https://gw.example.com"
const roundtripKey = "sk-roundtrip-key-1234567890"

type mergeKind struct {
	name                string
	configRelPath       []string
	buildJS             func() string
	buildPS             func() string
	ownedSubstrings     []string
	existingUnrelated   string
	unrelatedSubstrings []string
	malformedContent    string
}

func setupScriptMergeKinds() []mergeKind {
	claudeModels := map[string]string{
		"opus": "claude-opus-4", "sonnet": "claude-sonnet-4",
		"haiku": "claude-haiku-4", "subagent": "claude-haiku-4",
	}
	openCodeModels := map[string]string{"model": "gpt-4o", "small_model": "gpt-4o-mini"}
	openCodeAvailable := []string{"gpt-4o", "gpt-4o-mini"}
	piAvailable := []string{"gpt-4o", "gpt-4o-mini"}
	codexModels := map[string]string{"small": "gpt-5-mini", "medium": "gpt-5", "large": "gpt-5-pro"}
	ohMyPiAvailable := []string{"gpt-4o"}

	return []mergeKind{
		{
			name:          "claude-code",
			configRelPath: []string{".claude", "settings.json"},
			buildJS: func() string {
				return buildClaudeCodeScript(roundtripBaseURL, roundtripKey, claudeModels)
			},
			buildPS: func() string {
				return buildClaudeCodeScriptPS(roundtripBaseURL, roundtripKey, claudeModels)
			},
			ownedSubstrings: []string{roundtripBaseURL, roundtripKey, "claude-opus-4", "claude-sonnet-4", "claude-haiku-4"},
			existingUnrelated: `{
  "unrelatedTopLevel": "keep-me",
  "env": {
    "SOME_OTHER_VAR": "keep-me-too"
  }
}
`,
			unrelatedSubstrings: []string{"unrelatedTopLevel", "keep-me", "SOME_OTHER_VAR", "keep-me-too"},
			malformedContent:    "this is not valid json content",
		},
		{
			name:          "opencode",
			configRelPath: []string{".config", "opencode", "opencode.json"},
			buildJS: func() string {
				return buildOpenCodeScript(roundtripBaseURL, roundtripKey, openCodeModels, openCodeAvailable)
			},
			buildPS: func() string {
				return buildOpenCodeScriptPS(roundtripBaseURL, roundtripKey, openCodeModels, openCodeAvailable)
			},
			ownedSubstrings: []string{roundtripBaseURL, roundtripKey, "newapi/gpt-4o", "newapi/gpt-4o-mini", "@ai-sdk/openai-compatible"},
			existingUnrelated: `{
  "unrelatedTopLevel": "keep-me",
  "provider": {
    "otherprovider": { "npm": "other-npm-pkg" }
  }
}
`,
			unrelatedSubstrings: []string{"unrelatedTopLevel", "keep-me", "otherprovider", "other-npm-pkg"},
			malformedContent:    "this is not valid json content",
		},
		{
			name:          "pi",
			configRelPath: []string{".pi", "agent", "models.json"},
			buildJS: func() string {
				return buildPiScript(roundtripBaseURL, roundtripKey, piAvailable)
			},
			buildPS: func() string {
				return buildPiScriptPS(roundtripBaseURL, roundtripKey, piAvailable)
			},
			ownedSubstrings: []string{roundtripBaseURL, roundtripKey, "gpt-4o", "gpt-4o-mini"},
			existingUnrelated: `{
  "unrelatedTopLevel": "keep-me",
  "providers": {
    "otherprovider": { "baseUrl": "https://other.example.com" }
  }
}
`,
			unrelatedSubstrings: []string{"unrelatedTopLevel", "keep-me", "otherprovider", "https://other.example.com"},
			malformedContent:    "this is not valid json content",
		},
		{
			name:          "codex",
			configRelPath: []string{".codex", "config.toml"},
			buildJS: func() string {
				return buildCodexScript(roundtripBaseURL, roundtripKey, codexModels)
			},
			buildPS: func() string {
				return buildCodexScriptPS(roundtripBaseURL, roundtripKey, codexModels)
			},
			ownedSubstrings: []string{roundtripBaseURL + "/v1", "wire_api", "responses", "gpt-5-mini", "gpt-5", "gpt-5-pro"},
			existingUnrelated: `unrelated_key = "keep-me"

[model_providers.otherprovider]
name = "Other"
base_url = "https://other.example.com"
`,
			unrelatedSubstrings: []string{"unrelated_key", "keep-me", "[model_providers.otherprovider]", "https://other.example.com"},
			malformedContent: `model_provider = "openai"
this is not toml at all
`,
		},
		{
			name:          "oh-my-pi",
			configRelPath: []string{".omp", "agent", "models.yml"},
			buildJS: func() string {
				return buildOhMyPiScript(roundtripBaseURL, roundtripKey, ohMyPiAvailable)
			},
			buildPS: func() string {
				return buildOhMyPiScriptPS(roundtripBaseURL, roundtripKey, ohMyPiAvailable)
			},
			ownedSubstrings: []string{roundtripBaseURL, roundtripKey, "gpt-4o", "providers:"},
			existingUnrelated: `unrelated_top_level: keep-me
providers:
  otherprovider:
    baseUrl: https://other.example.com
    api: openai-completions
    apiKey: "other-key"
some_other_root_key: keep-me-too
`,
			unrelatedSubstrings: []string{"unrelated_top_level: keep-me", "otherprovider:", "https://other.example.com", "some_other_root_key: keep-me-too"},
			malformedContent: "providers:\n\totherprovider:\n    baseUrl: https://other.example.com\n",
		},
	}
}

// ---- generic round-trip scenarios, run for every application --------------

func TestSetupScriptMergeRoundTrip(t *testing.T) {
	for _, kind := range setupScriptMergeKinds() {
		kind := kind
		t.Run(kind.name, func(t *testing.T) {
			t.Run("node", func(t *testing.T) {
				if _, err := exec.LookPath("node"); err != nil {
					t.Skip("node not found on PATH; skipping")
				}
				runSetupScriptMergeScenarios(t, kind, kind.buildJS, runNodeMerge)
			})
			t.Run("powershell", func(t *testing.T) {
				if _, err := lookupPowerShellPath(); err != nil {
					t.Skip("powershell not found on PATH; skipping")
				}
				runSetupScriptMergeScenarios(t, kind, kind.buildPS, runPowerShellMerge)
			})
		})
	}
}

func runSetupScriptMergeScenarios(
	t *testing.T,
	kind mergeKind,
	build func() string,
	run func(t *testing.T, payload string, homeDir string) ([]byte, error),
) {
	t.Run("fresh file in missing directory", func(t *testing.T) {
		home := t.TempDir()
		out, err := run(t, build(), home)
		require.NoErrorf(t, err, "merge failed: %s", out)
		content := readConfigFile(t, home, kind.configRelPath)
		for _, s := range kind.ownedSubstrings {
			assert.Contains(t, content, s)
		}
	})

	t.Run("preserves unrelated existing content", func(t *testing.T) {
		home := t.TempDir()
		writeConfigFile(t, home, kind.configRelPath, kind.existingUnrelated)
		out, err := run(t, build(), home)
		require.NoErrorf(t, err, "merge failed: %s", out)
		content := readConfigFile(t, home, kind.configRelPath)
		for _, s := range kind.unrelatedSubstrings {
			assert.Contains(t, content, s, "unrelated existing content must survive verbatim")
		}
		for _, s := range kind.ownedSubstrings {
			assert.Contains(t, content, s)
		}
	})

	t.Run("re-running the same merge is idempotent", func(t *testing.T) {
		home := t.TempDir()
		out, err := run(t, build(), home)
		require.NoErrorf(t, err, "first merge failed: %s", out)
		first := readConfigFile(t, home, kind.configRelPath)

		out, err = run(t, build(), home)
		require.NoErrorf(t, err, "second merge failed: %s", out)
		second := readConfigFile(t, home, kind.configRelPath)

		assert.Equal(t, first, second, "re-running the same merge must be idempotent")
	})

	t.Run("a genuinely malformed file aborts and is left byte-identical", func(t *testing.T) {
		home := t.TempDir()
		writeConfigFile(t, home, kind.configRelPath, kind.malformedContent)
		out, err := run(t, build(), home)
		assert.Error(t, err, "a malformed existing file must abort the merge")
		content := readConfigFile(t, home, kind.configRelPath)
		assert.Equal(t, kind.malformedContent, content, "the file must be left byte-identical")
		fileName := kind.configRelPath[len(kind.configRelPath)-1]
		assert.Contains(t, string(out), fileName, "the error must name the file")
	})
}

// ---- Codex-specific: multi-line array + tab indentation --------------------

func TestCodexMergeAcceptsMultiLineArrayAndTabIndentation(t *testing.T) {
	// A real ~/.codex/config.toml entry for an MCP server: tab-indented,
	// with a multi-line array value. Both are ordinary valid TOML that the
	// original line-oriented editor rejected outright (see design.md).
	existing := "unrelated_key = \"keep-me\"\n" +
		"\n" +
		"[mcp_servers.everything]\n" +
		"\tcommand = \"npx\"\n" +
		"\targs = [\n" +
		"\t  \"-y\",\n" +
		"\t  \"everything\"\n" +
		"\t]\n"

	codexModels := map[string]string{"small": "gpt-5-mini", "medium": "gpt-5", "large": "gpt-5-pro"}
	relPath := []string{".codex", "config.toml"}

	assertMerged := func(t *testing.T, home string) {
		t.Helper()
		content := readConfigFile(t, home, relPath)
		assert.Contains(t, content, "[mcp_servers.everything]")
		assert.Contains(t, content, "args = [")
		assert.Contains(t, content, `"-y"`)
		assert.Contains(t, content, `"everything"`)
		assert.Contains(t, content, "unrelated_key")
		assert.Contains(t, content, "gpt-5")
	}

	t.Run("node", func(t *testing.T) {
		if _, err := exec.LookPath("node"); err != nil {
			t.Skip("node not found on PATH; skipping")
		}
		home := t.TempDir()
		writeConfigFile(t, home, relPath, existing)
		out, err := runNodeMerge(t, buildCodexScript(roundtripBaseURL, roundtripKey, codexModels), home)
		require.NoErrorf(t, err, "merge failed: %s", out)
		assertMerged(t, home)
	})

	t.Run("powershell", func(t *testing.T) {
		if _, err := lookupPowerShellPath(); err != nil {
			t.Skip("powershell not found on PATH; skipping")
		}
		home := t.TempDir()
		writeConfigFile(t, home, relPath, existing)
		out, err := runPowerShellMerge(t, buildCodexScriptPS(roundtripBaseURL, roundtripKey, codexModels), home)
		require.NoErrorf(t, err, "merge failed: %s", out)
		assertMerged(t, home)
	})
}

// ---- POSIX outer wrapper: mktemp portability and the node-presence guard --
//
// These two tests exercise renderPosixScript's own shell code (not just the
// Node.js payload it wraps) via a real `sh`, closing the gap on the two
// wrapper-level bugs this round covers: a macOS/BSD-incompatible mktemp
// template, and a missing, loud, early check for `node` on PATH.

// pathWithoutDir returns the PATH environment entries with dir removed,
// whatever separator the current PATH already uses (':' or ';'), so a
// subprocess can be made to genuinely fail to find an executable without
// guessing the platform's PATH format.
func pathWithoutDir(path string, dir string) string {
	sep := ":"
	if strings.Contains(path, ";") {
		sep = ";"
	}
	parts := strings.Split(path, sep)
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != dir && p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, sep)
}

func TestRenderPosixScriptRunsEndToEndViaRealMktemp(t *testing.T) {
	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found on PATH; skipping")
	}
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not found on PATH; skipping")
	}

	claudeModels := map[string]string{
		"opus": "claude-opus-4", "sonnet": "claude-sonnet-4",
		"haiku": "claude-haiku-4", "subagent": "claude-haiku-4",
	}
	jsPayload := buildClaudeCodeScript(roundtripBaseURL, roundtripKey, claudeModels)
	script := renderPosixScript(jsPayload, nil)

	home := t.TempDir()
	scriptPath := filepath.Join(t.TempDir(), "install.sh")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0o755))

	cmd := exec.Command(shPath, scriptPath)
	cmd.Env = isolatedHomeEnv(home)
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "script failed: %s", out)

	content := readConfigFile(t, home, []string{".claude", "settings.json"})
	assert.Contains(t, content, roundtripBaseURL)
	assert.Contains(t, content, roundtripKey)
}

func TestRenderPosixScriptFailsLoudlyWithoutNode(t *testing.T) {
	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found on PATH; skipping")
	}
	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not found on PATH; skipping")
	}

	claudeModels := map[string]string{
		"opus": "claude-opus-4", "sonnet": "claude-sonnet-4",
		"haiku": "claude-haiku-4", "subagent": "claude-haiku-4",
	}
	jsPayload := buildClaudeCodeScript(roundtripBaseURL, roundtripKey, claudeModels)
	script := renderPosixScript(jsPayload, nil)

	home := t.TempDir()
	scriptPath := filepath.Join(t.TempDir(), "install.sh")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0o755))

	env := isolatedHomeEnv(home)
	nodeDir := filepath.Dir(nodePath)
	for i, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			env[i] = "PATH=" + pathWithoutDir(strings.TrimPrefix(e, "PATH="), nodeDir)
		}
		if strings.HasPrefix(e, "Path=") {
			env[i] = "Path=" + pathWithoutDir(strings.TrimPrefix(e, "Path="), nodeDir)
		}
	}

	cmd := exec.Command(shPath, scriptPath)
	cmd.Env = env
	out, err := cmd.CombinedOutput()

	assert.Error(t, err, "the script must exit non-zero when node is missing")
	assert.Contains(t, strings.ToLower(string(out)), "node.js", "the failure must name Node.js as the requirement")
	assert.NoDirExists(t, filepath.Join(home, ".claude"), "nothing must be written before the node check")
}

// ---- POSIX outer wrapper: owned-export-only rc file must not false-abort ---
//
// renderPosixScript's rc-file guard used to abort with "refusing to replace
// ... with an empty file" whenever grep's filtered result came out empty,
// even when that emptiness was the correct, expected outcome of filtering
// out an rc file that consisted only of this script's own previously-written
// "export NEWAPI_API_KEY=" line (e.g. re-running the Codex setup a second
// time). That left the stale key line in place and exited 1 on a confusing
// "partial success". This must instead succeed and leave the rc file holding
// just the new export line.
func TestRenderPosixScriptReplacesRcFileThatIsOnlyTheOwnedExportLine(t *testing.T) {
	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found on PATH; skipping")
	}
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not found on PATH; skipping")
	}

	envVar := &setupScriptEnvVar{
		Name:    "NEWAPI_API_KEY",
		Value:   "new-key-0987654321",
		Comment: "Codex reads the API key only from this environment variable.",
	}
	// The JS payload itself is irrelevant to this rc-file-only regression;
	// keep it trivial so the test exercises only renderPosixScript's own
	// shell code around RC_FILE.
	script := renderPosixScript("console.log(\"noop\");\n", envVar)

	home := t.TempDir()
	rcPath := filepath.Join(home, ".bashrc")
	require.NoError(t, os.WriteFile(rcPath, []byte("export NEWAPI_API_KEY=old-key-1234567890\n"), 0o644))

	scriptPath := filepath.Join(t.TempDir(), "install.sh")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0o755))

	// Force the rc-file selection deterministically to .bashrc regardless of
	// the host's own $SHELL.
	env := make([]string, 0, len(os.Environ())+3)
	for _, e := range isolatedHomeEnv(home) {
		if strings.HasPrefix(e, "SHELL=") {
			continue
		}
		env = append(env, e)
	}
	env = append(env, "SHELL=/bin/bash")

	cmd := exec.Command(shPath, scriptPath)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "script must exit 0 when the rc file only ever held our own owned export line: %s", out)
	assert.NotContains(t, strings.ToLower(string(out)), "left unchanged", "a legitimately empty filter result must not be reported as a failed write")

	content, readErr := os.ReadFile(rcPath)
	require.NoError(t, readErr)
	assert.Equal(t, "export NEWAPI_API_KEY=\"new-key-0987654321\"\n", string(content), "the rc file must hold exactly the new export line")
}

// ---- PowerShell outer wrapper: malformed config shows a clean message -----
//
// renderPowerShellScript sets $ErrorActionPreference = "Stop", which turns
// the Write-Error calls in Read-JsonConfig's catch blocks into terminating
// errors — so the exit 1 that followed them never ran, and PowerShell's
// default host printed an exception-shaped stack trace (CategoryInfo /
// FullyQualifiedErrorId) instead of the intended clean one-line message.
// This asserts the user actually sees the clean message and a non-zero
// exit, with no such exception formatting, and that the malformed file is
// never overwritten.
func TestRenderPowerShellScriptShowsCleanMessageForMalformedConfig(t *testing.T) {
	psPath, err := lookupPowerShellPath()
	if err != nil {
		t.Skip("powershell not found on PATH; skipping")
	}

	claudeModels := map[string]string{
		"opus": "claude-opus-4", "sonnet": "claude-sonnet-4",
		"haiku": "claude-haiku-4", "subagent": "claude-haiku-4",
	}
	psPayload := buildClaudeCodeScriptPS(roundtripBaseURL, roundtripKey, claudeModels)
	script := renderPowerShellScript(psPayload, nil)

	home := t.TempDir()
	relPath := []string{".claude", "settings.json"}
	writeConfigFile(t, home, relPath, "this is not valid json content")

	scriptPath := filepath.Join(t.TempDir(), "install.ps1")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0o644))

	cmd := exec.Command(psPath, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
	cmd.Env = isolatedHomeEnv(home)
	out, err := cmd.CombinedOutput()

	assert.Error(t, err, "a malformed existing config file must exit non-zero")
	output := string(out)
	assert.Contains(t, output, "settings.json", "the clean error message must name the file")
	assert.Contains(t, output, "Could not parse existing configuration file", "the intended clean message must reach the user")
	assert.NotContains(t, output, "CategoryInfo", "the user must not see PowerShell's exception-shaped error formatting")
	assert.NotContains(t, output, "FullyQualifiedErrorId", "the user must not see PowerShell's exception-shaped error formatting")

	content := readConfigFile(t, home, relPath)
	assert.Equal(t, "this is not valid json content", content, "a malformed file must never be overwritten")
}
