package batata

import "fmt"

type Language int

const (
	LangMandarin   Language = 1
	LangSpanish    Language = 2
	LangEnglish    Language = 3
	LangHindi      Language = 4
	LangPortuguese Language = 5
	LangBengali    Language = 6
	LangRussian    Language = 7
	LangJapanese   Language = 8
	LangPunjabi    Language = 9
	LangVietnamese Language = 10
)

var SupportedLanguages = []string{
	"普通话 (Mandarin Chinese)",
	"Español (Spanish)",
	"English",
	"हिन्दी (Hindi)",
	"Português (Portuguese)",
	"বাংলা (Bengali)",
	"Русский (Russian)",
	"日本語 (Japanese)",
	"ਪੰਜਾਬੀ (Western Punjabi)",
	"Tiếng Việt (Vietnamese)",
}

type Strings struct {
	Intro          string
	ChooseLLM      string
	LLMOptions     string
	OllamaConfig   string
	CerebrasConfig string
	CerebrasKey    string
	CerebrasModel  string
	OllamaHost     string
	OllamaPort     string
	OllamaModel    string
	ConfigSaved    string
	MenuTitle      string
	MenuOptions    string
	KillGoodbye    string
	InvalidInput   string
	BackToBlady    string
	MiscConfig     string
	SetBrain       string
	BrainOffline   string
}

var LangStrings = map[Language]Strings{
	LangMandarin: {
		Intro:          "这是Batata，一个低功耗的简单核心，负责whatsabladerunner主核心Blady的基础设施。接下来是基本配置。",
		ChooseLLM:      "配置哪个LLM引擎？",
		LLMOptions:     "1. Ollama\n2. Cerebras\n3. 无",
		OllamaConfig:   "正在配置Ollama...",
		CerebrasConfig: "正在配置Cerebras。注意：联系信息和消息将发送到Cerebras。",
		CerebrasKey:    "请输入您的Cerebras开发者API密钥（有免费配额）：",
		CerebrasModel:  "输入Cerebras模型（建议：gpt-oss:120b）：",
		OllamaHost:     "输入Ollama主机（IP/URL）：",
		OllamaPort:     "输入Ollama端口（默认11434）：",
		OllamaModel:    "输入Ollama模型（建议：gpt-oss:20b或qwen30b+）：",
		ConfigSaved:    "配置已保存。需要更改配置时，只需说'Batata help'。",
		MenuTitle:      "Batata基本菜单：",
		MenuOptions:    "1. 更改语言\n2. 更新LLM配置\n3. 设置Blady的大脑\n4. 更新其他配置\n5. 终止whatsabladerunner\n6. 返回Blady",
		KillGoodbye:    "正在终止whatsabladerunner...再见！",
		InvalidInput:   "输入无效，请重试。",
		BackToBlady:    "将控制权交还给Blady。",
		MiscConfig:     "其他配置尚未实现。",
		SetBrain:       "选择Blady大脑使用哪个LLM：",
		BrainOffline:   "Blady的大脑已离线（未选择）。只能进行Batata核心交互。",
	},
	LangSpanish: {
		Intro:          "Este es Batata, el núcleo tonto de bajo esfuerzo que se encarga de la infraestructura básica para el núcleo principal de Blady en whatsabladerunner. A continuación, la configuración básica.",
		ChooseLLM:      "¿Qué motor LLM configurar?",
		LLMOptions:     "1. Ollama\n2. Cerebras\n3. Ninguno",
		OllamaConfig:   "Configurando Ollama...",
		CerebrasConfig: "Configurando Cerebras. Nota: La información de contacto y los mensajes se enviarán a Cerebras.",
		CerebrasKey:    "Por favor, introduce tu API Key de desarrollador de Cerebras (cuota gratuita disponible):",
		CerebrasModel:  "Introduce el Modelo de Cerebras (sugerido: gpt-oss:120b):",
		OllamaHost:     "Introduce el Host de Ollama (IP/URL):",
		OllamaPort:     "Introduce el Puerto de Ollama (por defecto 11434):",
		OllamaModel:    "Introduce el Modelo de Ollama (sugerido: gpt-oss:20b o qwen30b+):",
		ConfigSaved:    "Configuración guardada. Cuando necesites cambiar algo, di 'Batata help'.",
		MenuTitle:      "Menú Básico de Batata:",
		MenuOptions:    "1. Cambiar idioma\n2. Actualizar config LLM\n3. Configurar cerebro de Blady\n4. Actualizar config misc\n5. Matar whatsabladerunner\n6. Volver a Blady",
		KillGoodbye:    "Matando whatsabladerunner... ¡Adiós!",
		InvalidInput:   "Entrada inválida, intenta de nuevo.",
		BackToBlady:    "Devolviendo el control a Blady.",
		MiscConfig:     "Configuración miscelánea no implementada aún.",
		SetBrain:       "Elige qué LLM usar para el cerebro de Blady:",
		BrainOffline:   "El cerebro de Blady está OFFLINE (Ninguno seleccionado). Solo interacciones con Batata core son posibles.",
	},
	LangEnglish: {
		Intro:          "This is Batata, the low-effort dumb core that takes care of the basic infrastructure for whatsabladerunner main Blady's core. Basic configuration will follow.",
		ChooseLLM:      "Which LLM engine to config?",
		LLMOptions:     "1. Ollama\n2. Cerebras\n3. None",
		OllamaConfig:   "Configuring Ollama...",
		CerebrasConfig: "Configuring Cerebras. Note: Contact info and messages will be sent to Cerebras.",
		CerebrasKey:    "Please enter your Cerebras Developer API Key (free quota available):",
		CerebrasModel:  "Enter Cerebras Model (suggested: gpt-oss:120b):",
		OllamaHost:     "Enter Ollama Host (IP/URL):",
		OllamaPort:     "Enter Ollama Port (default 11434):",
		OllamaModel:    "Enter Ollama Model (suggested: gpt-oss:20b or qwen30b+):",
		ConfigSaved:    "Configuration saved. Anytime you need to change config, just say 'Batata help'.",
		MenuTitle:      "Batata Basic Menu:",
		MenuOptions:    "1. Change language\n2. Update LLM config\n3. Set Blady's brain\n4. Update misc config\n5. Kill whatsabladerunner\n6. Back to Blady",
		KillGoodbye:    "Killing whatsabladerunner... Bye!",
		InvalidInput:   "Invalid input, please try again.",
		BackToBlady:    "Returning control to Blady.",
		MiscConfig:     "Misc config not implemented yet.",
		SetBrain:       "Pick which LLM to use for Blady's brain:",
		BrainOffline:   "Blady's brain is OFFLINE (None selected). Only Batata core interactions are possible.",
	},
	LangHindi: {
		Intro:          "यह Batata है, एक कम-प्रयास वाला सरल कोर जो whatsabladerunner के मुख्य Blady कोर के लिए बुनियादी ढांचे का ध्यान रखता है। बुनियादी कॉन्फ़िगरेशन आगे होगा।",
		ChooseLLM:      "कौन सा LLM इंजन कॉन्फ़िगर करें?",
		LLMOptions:     "1. Ollama\n2. Cerebras\n3. कोई नहीं",
		OllamaConfig:   "Ollama कॉन्फ़िगर हो रहा है...",
		CerebrasConfig: "Cerebras कॉन्फ़िगर हो रहा है। नोट: संपर्क जानकारी और संदेश Cerebras को भेजे जाएंगे।",
		CerebrasKey:    "कृपया अपनी Cerebras डेवलपर API कुंजी दर्ज करें (मुफ्त कोटा उपलब्ध):",
		CerebrasModel:  "Cerebras मॉडल दर्ज करें (सुझाव: gpt-oss:120b):",
		OllamaHost:     "Ollama होस्ट दर्ज करें (IP/URL):",
		OllamaPort:     "Ollama पोर्ट दर्ज करें (डिफ़ॉल्ट 11434):",
		OllamaModel:    "Ollama मॉडल दर्ज करें (सुझाव: gpt-oss:20b या qwen30b+):",
		ConfigSaved:    "कॉन्फ़िगरेशन सहेजा गया। जब भी आपको कॉन्फ़िग बदलना हो, बस 'Batata help' कहें।",
		MenuTitle:      "Batata बेसिक मेनू:",
		MenuOptions:    "1. भाषा बदलें\n2. LLM कॉन्फ़िग अपडेट करें\n3. Blady का दिमाग सेट करें\n4. अन्य कॉन्फ़िग अपडेट करें\n5. whatsabladerunner बंद करें\n6. Blady पर वापस जाएं",
		KillGoodbye:    "whatsabladerunner बंद हो रहा है... अलविदा!",
		InvalidInput:   "अमान्य इनपुट, कृपया पुनः प्रयास करें।",
		BackToBlady:    "Blady को नियंत्रण वापस दे रहा है।",
		MiscConfig:     "अन्य कॉन्फ़िग अभी तक लागू नहीं हुआ।",
		SetBrain:       "Blady के दिमाग के लिए कौन सा LLM उपयोग करें:",
		BrainOffline:   "Blady का दिमाग ऑफ़लाइन है (कोई नहीं चुना गया)। केवल Batata कोर इंटरैक्शन संभव हैं।",
	},
	LangPortuguese: {
		Intro:          "Este é o Batata, o núcleo simples e de baixo esforço que cuida da infraestrutura básica do núcleo principal do Blady no whatsabladerunner. A configuração básica seguirá.",
		ChooseLLM:      "Qual motor LLM configurar?",
		LLMOptions:     "1. Ollama\n2. Cerebras\n3. Nenhum",
		OllamaConfig:   "Configurando Ollama...",
		CerebrasConfig: "Configurando Cerebras. Nota: Informações de contato e mensagens serão enviadas para o Cerebras.",
		CerebrasKey:    "Por favor, insira sua chave de API de desenvolvedor do Cerebras (cota gratuita disponível):",
		CerebrasModel:  "Insira o modelo Cerebras (sugerido: gpt-oss:120b):",
		OllamaHost:     "Insira o host do Ollama (IP/URL):",
		OllamaPort:     "Insira a porta do Ollama (padrão 11434):",
		OllamaModel:    "Insira o modelo Ollama (sugerido: gpt-oss:20b ou qwen30b+):",
		ConfigSaved:    "Configuração salva. Sempre que precisar alterar a config, basta dizer 'Batata help'.",
		MenuTitle:      "Menu Básico do Batata:",
		MenuOptions:    "1. Mudar idioma\n2. Atualizar config LLM\n3. Definir cérebro do Blady\n4. Atualizar config misc\n5. Encerrar whatsabladerunner\n6. Voltar ao Blady",
		KillGoodbye:    "Encerrando whatsabladerunner... Tchau!",
		InvalidInput:   "Entrada inválida, tente novamente.",
		BackToBlady:    "Devolvendo o controle ao Blady.",
		MiscConfig:     "Config miscelânea ainda não implementada.",
		SetBrain:       "Escolha qual LLM usar para o cérebro do Blady:",
		BrainOffline:   "O cérebro do Blady está OFFLINE (Nenhum selecionado). Apenas interações com o núcleo Batata são possíveis.",
	},
	LangBengali: {
		Intro:          "এটি Batata, একটি কম-প্রচেষ্টার সরল কোর যা whatsabladerunner-এর প্রধান Blady কোরের জন্য মৌলিক অবকাঠামো পরিচালনা করে। মৌলিক কনফিগারেশন অনুসরণ করবে।",
		ChooseLLM:      "কোন LLM ইঞ্জিন কনফিগার করবেন?",
		LLMOptions:     "1. Ollama\n2. Cerebras\n3. কোনোটিই নয়",
		OllamaConfig:   "Ollama কনফিগার করা হচ্ছে...",
		CerebrasConfig: "Cerebras কনফিগার করা হচ্ছে। দ্রষ্টব্য: যোগাযোগের তথ্য এবং বার্তা Cerebras-এ পাঠানো হবে।",
		CerebrasKey:    "অনুগ্রহ করে আপনার Cerebras ডেভেলপার API কী লিখুন (বিনামূল্যে কোটা উপলব্ধ):",
		CerebrasModel:  "Cerebras মডেল লিখুন (প্রস্তাবিত: gpt-oss:120b):",
		OllamaHost:     "Ollama হোস্ট লিখুন (IP/URL):",
		OllamaPort:     "Ollama পোর্ট লিখুন (ডিফল্ট 11434):",
		OllamaModel:    "Ollama মডেল লিখুন (প্রস্তাবিত: gpt-oss:20b বা qwen30b+):",
		ConfigSaved:    "কনফিগারেশন সংরক্ষিত হয়েছে। যখনই কনফিগ পরিবর্তন করতে চান, শুধু 'Batata help' বলুন।",
		MenuTitle:      "Batata মৌলিক মেনু:",
		MenuOptions:    "1. ভাষা পরিবর্তন করুন\n2. LLM কনফিগ আপডেট করুন\n3. Blady-এর মস্তিষ্ক সেট করুন\n4. অন্যান্য কনফিগ আপডেট করুন\n5. whatsabladerunner বন্ধ করুন\n6. Blady-তে ফিরে যান",
		KillGoodbye:    "whatsabladerunner বন্ধ হচ্ছে... বিদায়!",
		InvalidInput:   "অবৈধ ইনপুট, পুনরায় চেষ্টা করুন।",
		BackToBlady:    "Blady-কে নিয়ন্ত্রণ ফিরিয়ে দেওয়া হচ্ছে।",
		MiscConfig:     "অন্যান্য কনফিগ এখনও বাস্তবায়িত হয়নি।",
		SetBrain:       "Blady-এর মস্তিষ্কের জন্য কোন LLM ব্যবহার করবেন:",
		BrainOffline:   "Blady-এর মস্তিষ্ক অফলাইন (কোনোটি নির্বাচিত নয়)। শুধুমাত্র Batata কোর ইন্টারঅ্যাকশন সম্ভব।",
	},
	LangRussian: {
		Intro:          "Это Batata, простое ядро с минимальными усилиями, которое заботится о базовой инфраструктуре основного ядра Blady в whatsabladerunner. Далее следует базовая настройка.",
		ChooseLLM:      "Какой LLM-движок настроить?",
		LLMOptions:     "1. Ollama\n2. Cerebras\n3. Нет",
		OllamaConfig:   "Настройка Ollama...",
		CerebrasConfig: "Настройка Cerebras. Примечание: контактная информация и сообщения будут отправлены в Cerebras.",
		CerebrasKey:    "Пожалуйста, введите ваш API-ключ разработчика Cerebras (доступна бесплатная квота):",
		CerebrasModel:  "Введите модель Cerebras (рекомендуется: gpt-oss:120b):",
		OllamaHost:     "Введите хост Ollama (IP/URL):",
		OllamaPort:     "Введите порт Ollama (по умолчанию 11434):",
		OllamaModel:    "Введите модель Ollama (рекомендуется: gpt-oss:20b или qwen30b+):",
		ConfigSaved:    "Конфигурация сохранена. Когда нужно изменить настройки, просто скажите 'Batata help'.",
		MenuTitle:      "Базовое меню Batata:",
		MenuOptions:    "1. Сменить язык\n2. Обновить конфиг LLM\n3. Настроить мозг Blady\n4. Обновить прочие настройки\n5. Завершить whatsabladerunner\n6. Вернуться к Blady",
		KillGoodbye:    "Завершаем whatsabladerunner... Пока!",
		InvalidInput:   "Неверный ввод, попробуйте снова.",
		BackToBlady:    "Возвращаем управление Blady.",
		MiscConfig:     "Прочие настройки пока не реализованы.",
		SetBrain:       "Выберите, какой LLM использовать для мозга Blady:",
		BrainOffline:   "Мозг Blady ОФЛАЙН (ничего не выбрано). Возможны только взаимодействия с ядром Batata.",
	},
	LangJapanese: {
		Intro:          "これはBatataです。whatsabladerunnerのメインBladyコアの基本インフラを担当する低負荷のシンプルなコアです。基本設定が続きます。",
		ChooseLLM:      "どのLLMエンジンを設定しますか？",
		LLMOptions:     "1. Ollama\n2. Cerebras\n3. なし",
		OllamaConfig:   "Ollamaを設定中...",
		CerebrasConfig: "Cerebrasを設定中。注意：連絡先情報とメッセージはCerebrasに送信されます。",
		CerebrasKey:    "Cerebras開発者APIキーを入力してください（無料枠あり）：",
		CerebrasModel:  "Cerebrasモデルを入力（推奨：gpt-oss:120b）：",
		OllamaHost:     "Ollamaホストを入力（IP/URL）：",
		OllamaPort:     "Ollamaポートを入力（デフォルト11434）：",
		OllamaModel:    "Ollamaモデルを入力（推奨：gpt-oss:20bまたはqwen30b+）：",
		ConfigSaved:    "設定が保存されました。設定を変更したい時は'Batata help'と言ってください。",
		MenuTitle:      "Batata基本メニュー：",
		MenuOptions:    "1. 言語を変更\n2. LLM設定を更新\n3. Bladyの脳を設定\n4. その他の設定を更新\n5. whatsabladerunnerを終了\n6. Bladyに戻る",
		KillGoodbye:    "whatsabladerunnerを終了しています...さようなら！",
		InvalidInput:   "無効な入力です。もう一度お試しください。",
		BackToBlady:    "Bladyに制御を戻しています。",
		MiscConfig:     "その他の設定はまだ実装されていません。",
		SetBrain:       "Bladyの脳に使用するLLMを選択：",
		BrainOffline:   "Bladyの脳はオフラインです（未選択）。Batataコアとの対話のみ可能です。",
	},
	LangPunjabi: {
		Intro:          "ਇਹ Batata ਹੈ, ਇੱਕ ਘੱਟ-ਮਿਹਨਤ ਵਾਲਾ ਸਧਾਰਨ ਕੋਰ ਜੋ whatsabladerunner ਦੇ ਮੁੱਖ Blady ਕੋਰ ਲਈ ਬੁਨਿਆਦੀ ਢਾਂਚੇ ਦਾ ਧਿਆਨ ਰੱਖਦਾ ਹੈ। ਬੁਨਿਆਦੀ ਸੰਰਚਨਾ ਅੱਗੇ ਹੋਵੇਗੀ।",
		ChooseLLM:      "ਕਿਹੜਾ LLM ਇੰਜਣ ਸੰਰਚਿਤ ਕਰਨਾ ਹੈ?",
		LLMOptions:     "1. Ollama\n2. Cerebras\n3. ਕੋਈ ਨਹੀਂ",
		OllamaConfig:   "Ollama ਸੰਰਚਿਤ ਹੋ ਰਿਹਾ ਹੈ...",
		CerebrasConfig: "Cerebras ਸੰਰਚਿਤ ਹੋ ਰਿਹਾ ਹੈ। ਨੋਟ: ਸੰਪਰਕ ਜਾਣਕਾਰੀ ਅਤੇ ਸੁਨੇਹੇ Cerebras ਨੂੰ ਭੇਜੇ ਜਾਣਗੇ।",
		CerebrasKey:    "ਕਿਰਪਾ ਕਰਕੇ ਆਪਣੀ Cerebras ਡਿਵੈਲਪਰ API ਕੁੰਜੀ ਦਾਖਲ ਕਰੋ (ਮੁਫ਼ਤ ਕੋਟਾ ਉਪਲਬਧ):",
		CerebrasModel:  "Cerebras ਮਾਡਲ ਦਾਖਲ ਕਰੋ (ਸੁਝਾਅ: gpt-oss:120b):",
		OllamaHost:     "Ollama ਹੋਸਟ ਦਾਖਲ ਕਰੋ (IP/URL):",
		OllamaPort:     "Ollama ਪੋਰਟ ਦਾਖਲ ਕਰੋ (ਮੂਲ 11434):",
		OllamaModel:    "Ollama ਮਾਡਲ ਦਾਖਲ ਕਰੋ (ਸੁਝਾਅ: gpt-oss:20b ਜਾਂ qwen30b+):",
		ConfigSaved:    "ਸੰਰਚਨਾ ਸੁਰੱਖਿਅਤ ਹੋ ਗਈ। ਜਦੋਂ ਵੀ ਸੰਰਚਨਾ ਬਦਲਣੀ ਹੋਵੇ, ਬੱਸ 'Batata help' ਕਹੋ।",
		MenuTitle:      "Batata ਬੁਨਿਆਦੀ ਮੀਨੂ:",
		MenuOptions:    "1. ਭਾਸ਼ਾ ਬਦਲੋ\n2. LLM ਸੰਰਚਨਾ ਅੱਪਡੇਟ ਕਰੋ\n3. Blady ਦਾ ਦਿਮਾਗ ਸੈੱਟ ਕਰੋ\n4. ਹੋਰ ਸੰਰਚਨਾ ਅੱਪਡੇਟ ਕਰੋ\n5. whatsabladerunner ਬੰਦ ਕਰੋ\n6. Blady 'ਤੇ ਵਾਪਸ ਜਾਓ",
		KillGoodbye:    "whatsabladerunner ਬੰਦ ਹੋ ਰਿਹਾ ਹੈ... ਅਲਵਿਦਾ!",
		InvalidInput:   "ਅਵੈਧ ਇਨਪੁੱਟ, ਕਿਰਪਾ ਕਰਕੇ ਦੁਬਾਰਾ ਕੋਸ਼ਿਸ਼ ਕਰੋ।",
		BackToBlady:    "Blady ਨੂੰ ਕੰਟਰੋਲ ਵਾਪਸ ਕਰ ਰਿਹਾ ਹੈ।",
		MiscConfig:     "ਹੋਰ ਸੰਰਚਨਾ ਅਜੇ ਲਾਗੂ ਨਹੀਂ ਹੋਈ।",
		SetBrain:       "Blady ਦੇ ਦਿਮਾਗ ਲਈ ਕਿਹੜਾ LLM ਵਰਤਣਾ ਹੈ:",
		BrainOffline:   "Blady ਦਾ ਦਿਮਾਗ ਆਫਲਾਈਨ ਹੈ (ਕੋਈ ਨਹੀਂ ਚੁਣਿਆ)। ਸਿਰਫ਼ Batata ਕੋਰ ਇੰਟਰੈਕਸ਼ਨ ਸੰਭਵ ਹਨ।",
	},
	LangVietnamese: {
		Intro:          "Đây là Batata, lõi đơn giản ít nỗ lực chịu trách nhiệm về cơ sở hạ tầng cơ bản cho lõi Blady chính của whatsabladerunner. Cấu hình cơ bản sẽ theo sau.",
		ChooseLLM:      "Cấu hình công cụ LLM nào?",
		LLMOptions:     "1. Ollama\n2. Cerebras\n3. Không có",
		OllamaConfig:   "Đang cấu hình Ollama...",
		CerebrasConfig: "Đang cấu hình Cerebras. Lưu ý: Thông tin liên hệ và tin nhắn sẽ được gửi đến Cerebras.",
		CerebrasKey:    "Vui lòng nhập API Key nhà phát triển Cerebras của bạn (có hạn ngạch miễn phí):",
		CerebrasModel:  "Nhập mô hình Cerebras (đề xuất: gpt-oss:120b):",
		OllamaHost:     "Nhập máy chủ Ollama (IP/URL):",
		OllamaPort:     "Nhập cổng Ollama (mặc định 11434):",
		OllamaModel:    "Nhập mô hình Ollama (đề xuất: gpt-oss:20b hoặc qwen30b+):",
		ConfigSaved:    "Đã lưu cấu hình. Bất cứ khi nào bạn cần thay đổi cấu hình, chỉ cần nói 'Batata help'.",
		MenuTitle:      "Menu Cơ Bản Batata:",
		MenuOptions:    "1. Đổi ngôn ngữ\n2. Cập nhật cấu hình LLM\n3. Đặt não của Blady\n4. Cập nhật cấu hình khác\n5. Dừng whatsabladerunner\n6. Quay lại Blady",
		KillGoodbye:    "Đang dừng whatsabladerunner... Tạm biệt!",
		InvalidInput:   "Đầu vào không hợp lệ, vui lòng thử lại.",
		BackToBlady:    "Trả quyền điều khiển cho Blady.",
		MiscConfig:     "Cấu hình khác chưa được triển khai.",
		SetBrain:       "Chọn LLM nào để sử dụng cho não của Blady:",
		BrainOffline:   "Não của Blady đang OFFLINE (Không chọn). Chỉ có thể tương tác với lõi Batata.",
	},
}

func GetString(lang Language, selector func(Strings) string) string {
	strs, ok := LangStrings[lang]
	if !ok {
		strs = LangStrings[LangEnglish]
	}
	return selector(strs)
}

func LangSelectMenu() string {
	menu := ""
	for i, l := range SupportedLanguages {
		menu += fmt.Sprintf("%d. %s\n", i+1, l)
	}
	return menu
}
