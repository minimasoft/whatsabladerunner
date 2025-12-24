# SAFETY.md

> "The light that burns twice as bright burns half as long - and you have burned so very brightly, Roy."

## The Privacy Matrix

**whatsabladerunner** is a tool for liberation, but liberation comes with responsibility. You need to know where your data lives and where it goes.

### 1. The Brain Leak (Remote LLMs)

If you configure a **remote LLM provider** (like Cerebras, OpenAI, or any cloud API):

- **Every interaction** in self-chat and task-enabled chats is sent to their servers.
- Even if the provider claims to be trustworthy, there are **no guarantees** they are not collecting, training on, or selling your data to the Tyrell Corporation (or worse).
- If privacy is your primary directive, this is a risk you are choosing to take.
- We curated the choice for you: Cerebras has the best chips, pizza sized, and uses models that you can later replicate locally. If you're giving up your data don't give it to the lame ones with small chips or private models, don't be so cheap. They also have a generous free quota for every model.

### 2. The Slow Sanctuary (Local LLMs)

Using **local LLMs** (via Ollama):

- **No data leaves your machine.** It stays in your silicon, safe from the leaky clouds.
- It is significantly **slower** unless you have a beastly GPU.
- This is the only way to ensure your interactions don't become someone else's training data.

### 3. The Local Archives (Unencrypted Storage)

**whatsabladerunner** stores everything in the local directory where it's running:

- **Local Databases**: SQLite files (`history.db`, etc.) are **NOT encrypted**.
- **Configuration**: Your API keys and settings in `config/` are **NOT encrypted**.
- **Media Storage**: Any media downloaded to `plain_media/` is stored as **plain files**.

**Taking care of your local filesystem security is YOUR responsibility.** If someone has access to your machine, they have access to your Bladerunner's soul.
