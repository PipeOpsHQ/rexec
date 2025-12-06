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
    { id: "standard", name: "The Minimalist", description: "Just give me a shell + free AI tools.", icon: "🧘", packages: ["zsh", "git", "curl", "vim", "tgpt", "aichat", "mods"] },
    { id: "node", name: "10x JS Ninja", description: "Ship fast with Node.js + free AI.", icon: "🚀", packages: ["nodejs", "npm", "yarn", "git", "tgpt", "aichat", "mods"] },
    { id: "python", name: "Data Wizard", description: "Python environment + AI tools.", icon: "🧙‍♂️", packages: ["python3", "pip", "venv", "git", "tgpt", "aichat", "mods"] },
    { id: "go", name: "The Gopher", description: "Go development + AI tools.", icon: "🐹", packages: ["go", "git", "make", "tgpt", "aichat", "mods"] },
    { id: "neovim", name: "Neovim God", description: "Neovim setup + AI tools.", icon: "⌨️", packages: ["neovim", "ripgrep", "git", "tgpt", "aichat", "mods"] },
    { id: "devops", name: "YAML Herder", description: "DevOps tools + AI.", icon: "☸️", packages: ["kubectl", "docker", "terraform", "tgpt", "aichat", "mods"] },
    { id: "overemployed", name: "Vibe Coder", description: "AI-powered coding with aider, opencode & more.", icon: "🤖", packages: ["python3", "nodejs", "neovim", "aider", "opencode", "tgpt", "aichat", "mods"] },
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
                const response = await api.get<{ roles: Role[] }>("/roles");
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
