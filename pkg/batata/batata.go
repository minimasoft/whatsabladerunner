package batata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"go.mau.fi/whatsmeow/types"
)

// State enum
type State int

const (
	StateIdle           State = 0
	StateSetupLanguage  State = 1
	StateSetupIntro     State = 2 // Transition state, shows intro then asks LLM
	StateSetupLLMChoice State = 3
	StateSetupOllama    State = 4 // Sub-states for Ollama could be handled or just generic input
	StateSetupCerebras  State = 5
	StateMainMenu       State = 6
	StateConfigMisc     State = 7
	StateSetBrain       State = 8
)

// Sub-states or contextual variables might be needed
// For simplicity, we'll store "NextStep" string or similar if deeply nested,
// but for this flow, flat states + a few variables work.

type Config struct {
	Language       Language `json:"language"`
	LLMProvider    string   `json:"llm_provider"` // "ollama", "cerebras", "none"
	OllamaHost     string   `json:"ollama_host"`
	OllamaPort     string   `json:"ollama_port"`
	OllamaModel    string   `json:"ollama_model"`
	CerebrasKey    string   `json:"cerebras_key"`
	CerebrasModel  string   `json:"cerebras_model"`
	BrainProvider  string   `json:"brain_provider"` // Which one is active for Blady
	TimeoutSeconds int      `json:"timeout_seconds"`
}

type Kernel struct {
	ConfigPath string
	Config     Config
	State      State
	StateMu    sync.Mutex

	// Temporary state for multi-step inputs
	tempInputStep int
}

func NewKernel(configDir string) *Kernel {
	return &Kernel{
		ConfigPath: filepath.Join(configDir, "batata.json"),
		Config: Config{
			Language:       LangEnglish, // Default fallack
			LLMProvider:    "none",
			BrainProvider:  "none",
			TimeoutSeconds: 60,
			OllamaHost:     "http://localhost",
			OllamaPort:     "11434",
			OllamaModel:    "qwen3:8b",
			CerebrasModel:  "gpt-oss-120b",
		},
		State: StateIdle,
	}
}

func (k *Kernel) Load() error {
	data, err := os.ReadFile(k.ConfigPath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &k.Config)
}

func (k *Kernel) Save() error {
	data, err := json.MarshalIndent(k.Config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(k.ConfigPath, data, 0644)
}

func (k *Kernel) StartSetup(sendFunc func(string)) {
	k.StateMu.Lock()
	defer k.StateMu.Unlock()

	k.State = StateSetupLanguage
	sendFunc(LangSelectMenu())
}

// HandleMessage returns true if the message was consumed by Batata
func (k *Kernel) HandleMessage(msgText string, senderJID, chatJID types.JID, sendFunc func(string), killFunc func()) bool {
	// Only handle if message is "Batata help" OR we are in a non-idle state
	// AND it is a self-chat (sender == chat.User? or just explicitly passed as trusted)
	// For "Note to Self" check:
	isSelfChat := (senderJID.User == chatJID.User)

	// If not self chat, ignore completely (unless we want to support admin from elsewhere, but requirement says self-chat)
	if !isSelfChat {
		return false
	}

	cleanMsg := strings.TrimSpace(msgText)

	k.StateMu.Lock()
	defer k.StateMu.Unlock()

	// Activation trigger
	if strings.EqualFold(cleanMsg, "Batata help") {
		k.State = StateMainMenu
		k.sendMenu(sendFunc)
		return true
	}

	if k.State == StateIdle {
		return false
	}

	// Process input based on state
	switch k.State {
	case StateSetupLanguage:
		// Expecting 1-10
		choice, err := strconv.Atoi(cleanMsg)
		if err != nil || choice < 1 || choice > 10 {
			sendFunc("Invalid choice. / Opción inválida. \n" + LangSelectMenu())
			return true
		}
		k.Config.Language = Language(choice)
		// Proceed to Intro
		k.State = StateSetupLLMChoice

		intro := k.s(func(s Strings) string { return s.Intro })
		prompt := k.s(func(s Strings) string { return s.ChooseLLM + "\n" + s.LLMOptions })

		sendFunc(intro + "\n\n" + prompt)
		return true

	case StateSetupLLMChoice:
		switch cleanMsg {
		case "1": // Ollama
			k.Config.LLMProvider = "ollama"
			k.Config.BrainProvider = "ollama"
			k.State = StateSetupOllama
			k.tempInputStep = 0
			sendFunc(k.s(func(s Strings) string { return s.OllamaHost }))
		case "2": // Cerebras
			k.Config.LLMProvider = "cerebras"
			k.Config.BrainProvider = "cerebras"
			k.State = StateSetupCerebras
			k.tempInputStep = 0
			sendFunc(k.s(func(s Strings) string { return s.CerebrasConfig + "\n" + s.CerebrasKey }))
		case "3": // None
			k.Config.LLMProvider = "none"
			k.Config.BrainProvider = "none"
			k.finishConfig(sendFunc)
		default:
			sendFunc(k.s(func(s Strings) string { return s.InvalidInput }))
		}
		return true

	case StateSetupOllama:
		// Step 0: Host, 1: Port, 2: Model
		switch k.tempInputStep {
		case 0:
			k.Config.OllamaHost = cleanMsg
			k.tempInputStep++
			sendFunc(k.s(func(s Strings) string { return s.OllamaPort }))
		case 1:
			k.Config.OllamaPort = cleanMsg
			k.tempInputStep++
			sendFunc(k.s(func(s Strings) string { return s.OllamaModel }))
		case 2:
			k.Config.OllamaModel = cleanMsg
			k.finishConfig(sendFunc)
		}
		return true

	case StateSetupCerebras:
		// Step 0: Key, 1: Model
		switch k.tempInputStep {
		case 0:
			k.Config.CerebrasKey = cleanMsg
			k.tempInputStep++
			sendFunc(k.s(func(s Strings) string { return s.CerebrasModel }))
		case 1:
			k.Config.CerebrasModel = cleanMsg
			k.finishConfig(sendFunc)
		}
		return true

	case StateMainMenu:
		switch cleanMsg {
		case "1": // Change Language
			k.State = StateSetupLanguage
			sendFunc(LangSelectMenu())
		case "2": // Update LLM Config
			k.State = StateSetupLLMChoice
			sendFunc(k.s(func(s Strings) string { return s.ChooseLLM + "\n" + s.LLMOptions }))
		case "3": // Set Brain
			k.State = StateSetBrain
			sendFunc(k.s(func(s Strings) string { return s.SetBrain + "\n1. Ollama\n2. Cerebras\n3. None" }))
		case "4": // Misc
			sendFunc(k.s(func(s Strings) string { return s.MiscConfig }))
			k.sendMenu(sendFunc) // Return to menu
		case "5": // Kill
			sendFunc(k.s(func(s Strings) string { return s.KillGoodbye }))
			if killFunc != nil {
				killFunc()
			}
		case "6": // Back to Blady
			k.State = StateIdle
			sendFunc(k.s(func(s Strings) string { return s.BackToBlady }))
		default:
			sendFunc(k.s(func(s Strings) string { return s.InvalidInput }))
			k.sendMenu(sendFunc)
		}
		return true

	case StateSetBrain:
		switch cleanMsg {
		case "1":
			k.Config.BrainProvider = "ollama"
			sendFunc("Brain set to Ollama.")
		case "2":
			k.Config.BrainProvider = "cerebras"
			sendFunc("Brain set to Cerebras.")
		case "3":
			k.Config.BrainProvider = "none"
			sendFunc("Brain set to None (Offline).")
		}
		k.Save()
		k.State = StateMainMenu
		k.sendMenu(sendFunc)
		return true
	}

	return true
}

func (k *Kernel) s(selector func(Strings) string) string {
	return GetString(k.Config.Language, selector)
}

func (k *Kernel) sendMenu(sendFunc func(string)) {
	sendFunc(k.s(func(s Strings) string { return s.MenuTitle + "\n" + s.MenuOptions }))
}

func (k *Kernel) finishConfig(sendFunc func(string)) {
	k.Save()
	// Warn if brain is offline
	if k.Config.BrainProvider == "none" {
		sendFunc(k.s(func(s Strings) string { return s.BrainOffline }))
	}
	sendFunc(k.s(func(s Strings) string { return s.ConfigSaved }))
	k.State = StateIdle
}
