const FALLBACK_SITE_URL = "https://rexec.sh";

const defaultAllowedOrigins = [
  "https://rexec.pipeops.app",
  "https://rexec.sh",
  "https://rexec.pipeops.sh",
  "https://rexec.io",
  "https://rexec.sh",
  "https://rexec.cloud",
  "http://localhost:3000",
  "http://localhost:4173",
  "http://localhost:5173",
  "http://localhost:8080",
  "http://127.0.0.1:3000",
  "http://127.0.0.1:4173",
  "http://127.0.0.1:5173",
  "http://127.0.0.1:8080",
];

type NullableString = string | null | undefined;

function normalizeOrigin(origin: NullableString): string | null {
  if (!origin) return null;
  try {
    const url = new URL(origin.trim());
    return `${url.protocol}//${url.host}`;
  } catch {
    return null;
  }
}

const envAllowedOrigins = (import.meta.env.VITE_ALLOWED_ORIGINS || "")
  .split(",")
  .map((value: NullableString) => normalizeOrigin(value))
  .filter((value: any): value is string => Boolean(value));

const envSiteUrl = normalizeOrigin(import.meta.env.VITE_SITE_URL);
const fallbackOrigin = normalizeOrigin(FALLBACK_SITE_URL)!;

const allowedOriginsSet = new Set<string>();
for (const origin of defaultAllowedOrigins) {
  const normalized = normalizeOrigin(origin);
  if (normalized) allowedOriginsSet.add(normalized);
}
for (const origin of envAllowedOrigins) {
  allowedOriginsSet.add(origin);
}
allowedOriginsSet.add(envSiteUrl || fallbackOrigin);

const allowedOrigins = Array.from(allowedOriginsSet);

function sanitizePath(path?: string): string {
  if (!path || path === "/") return "/";
  return path.startsWith("/") ? path : `/${path}`;
}

export function getAllowedOrigins(): string[] {
  return allowedOrigins;
}

export function isAllowedOrigin(origin: NullableString): boolean {
  const normalized = normalizeOrigin(origin);
  return normalized ? allowedOriginsSet.has(normalized) : false;
}

export function getCanonicalOrigin(currentOrigin?: NullableString): string {
  const normalizedCurrent = normalizeOrigin(currentOrigin);
  if (normalizedCurrent && allowedOriginsSet.has(normalizedCurrent)) {
    return normalizedCurrent;
  }

  if (typeof window !== "undefined") {
    const windowOrigin = normalizeOrigin(window.location.origin);
    if (windowOrigin && allowedOriginsSet.has(windowOrigin)) {
      return windowOrigin;
    }
  }

  return envSiteUrl || fallbackOrigin;
}

export function buildCanonicalUrl(path?: string, origin?: NullableString): string {
  const canonicalOrigin = getCanonicalOrigin(origin);
  return `${canonicalOrigin}${sanitizePath(path)}`;
}

export function getAssetUrl(assetPath: string): string {
  const path = assetPath.startsWith("/") ? assetPath : `/${assetPath}`;
  return `${getCanonicalOrigin()}${path}`;
}

function upsertMeta(selector: string, create: () => HTMLElement): HTMLElement {
  const existing = document.querySelector(selector);
  if (existing) return existing as HTMLElement;
  const element = create();
  document.head.appendChild(element);
  return element;
}

export function syncCanonicalTags(pathname?: string) {
  if (typeof document === "undefined") return;
  const canonicalUrl = buildCanonicalUrl(pathname || (typeof window !== "undefined" ? window.location.pathname : "/"));
  const canonicalLink = upsertMeta('link[rel="canonical"]', () => {
    const link = document.createElement("link");
    link.setAttribute("rel", "canonical");
    return link;
  }) as HTMLLinkElement;
  canonicalLink.setAttribute("href", canonicalUrl);

  const ogUrl = upsertMeta('meta[property="og:url"]', () => {
    const meta = document.createElement("meta");
    meta.setAttribute("property", "og:url");
    return meta;
  }) as HTMLMetaElement;
  ogUrl.setAttribute("content", canonicalUrl);

  const twitterUrl = upsertMeta('meta[name="twitter:url"]', () => {
    const meta = document.createElement("meta");
    meta.setAttribute("name", "twitter:url");
    return meta;
  }) as HTMLMetaElement;
  twitterUrl.setAttribute("content", canonicalUrl);
}

export function syncSocialImageTags(assetPath = "/og-image.png") {
  if (typeof document === "undefined") return;
  const assetUrl = getAssetUrl(assetPath);

  const ogImage = upsertMeta('meta[property="og:image"]', () => {
    const meta = document.createElement("meta");
    meta.setAttribute("property", "og:image");
    return meta;
  }) as HTMLMetaElement;
  ogImage.setAttribute("content", assetUrl);

  const twitterImage = upsertMeta('meta[name="twitter:image"]', () => {
    const meta = document.createElement("meta");
    meta.setAttribute("name", "twitter:image");
    return meta;
  }) as HTMLMetaElement;
  twitterImage.setAttribute("content", assetUrl);
}

export function updateInitialCanonicalTags() {
  const pathname = typeof window !== "undefined" ? window.location.pathname : "/";
  syncCanonicalTags(pathname);
  syncSocialImageTags();
}
