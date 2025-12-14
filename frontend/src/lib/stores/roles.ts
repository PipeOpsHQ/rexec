import { writable } from 'svelte/store';
import { api } from '$utils/api';

export interface Role {
    id: string;
    name: string;
    description: string;
    icon: string;
    packages: string[];
}

// Default roles to show while loading or on error
export const defaultRoles: Role[] = [
    { id: "standard", name: "The Minimalist", description: "I use Arch btw. Just give me a shell + free AI tools.", icon: "terminal", packages: ["zsh", "git", "curl", "wget", "vim", "nano", "htop", "jq", "neofetch", "tgpt", "aichat", "mods", "zsh-autosuggestions", "zsh-syntax-highlighting"] },
    { id: "node", name: "10x JS Ninja", description: "Ship fast, break things, npm install everything + free AI.", icon: "nodejs", packages: ["zsh", "git", "nodejs", "npm", "yarn", "tgpt", "aichat", "mods", "zsh-autosuggestions", "zsh-syntax-highlighting"] },
    { id: "python", name: "Data Wizard", description: "Import antigravity. I speak in list comprehensions + AI.", icon: "python", packages: ["zsh", "git", "python3", "python3-pip", "python3-venv", "tgpt", "aichat", "mods", "zsh-autosuggestions", "zsh-syntax-highlighting"] },
    { id: "go", name: "The Gopher", description: "If err != nil { panic(err) }. Simplicity + AI tools.", icon: "golang", packages: ["zsh", "git", "make", "go", "tgpt", "aichat", "mods", "zsh-autosuggestions", "zsh-syntax-highlighting"] },
    { id: "neovim", name: "Neovim God", description: "My config is longer than your code. Mouse? AI helps.", icon: "edit", packages: ["zsh", "git", "neovim", "ripgrep", "gcc", "make", "curl", "tgpt", "aichat", "mods", "zsh-autosuggestions", "zsh-syntax-highlighting"] },
    { id: "devops", name: "YAML Herder", description: "I don't write code, I write config. AI assists.", icon: "devops", packages: ["zsh", "git", "docker", "kubectl", "ansible", "terraform", "tgpt", "aichat", "mods", "zsh-autosuggestions", "zsh-syntax-highlighting"] },
    { id: "overemployed", name: "Vibe Coder", description: "AI-powered coding: tgpt, aichat, mods, aider, opencode & more.", icon: "ai", packages: ["zsh", "git", "tmux", "python3", "python3-pip", "python3-venv", "nodejs", "npm", "curl", "wget", "htop", "vim", "neovim", "ripgrep", "fzf", "jq", "tgpt", "aichat", "mods", "aider", "opencode", "llm", "sgpt", "zsh-autosuggestions", "zsh-syntax-highlighting"] },
];

function createRolesStore() {
    const { subscribe, set, update } = writable<{ roles: Role[], loading: boolean, error: string | null }>({
        roles: [],
        loading: false,
        error: null
    });

    let loaded = false;

    return {
        subscribe,
        load: async (force = false) => {
            if (loaded && !force) return;

            update(s => ({ ...s, loading: true, error: null }));

            try {
                const response = await api.get<{ roles: Role[] }>("/api/roles");
                if (response.data?.roles && Array.isArray(response.data.roles) && response.data.roles.length > 0) {
                    set({ roles: response.data.roles, loading: false, error: null });
                    loaded = true;
                } else {
                    // Fallback
                    set({ roles: defaultRoles, loading: false, error: null }); // Don't treat as error, just use defaults
                }
            } catch (e) {
                console.error("Failed to load roles:", e);
                set({ roles: defaultRoles, loading: false, error: e instanceof Error ? e.message : "Failed to load roles" });
            }
        }
    };
}

export const roles = createRolesStore();
