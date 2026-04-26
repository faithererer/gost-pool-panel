import { Settings } from "../api/types";

export type ProxyProtocol = "http" | "socks5h";

export function hostFromBaseURL(baseURL: string, fallback: string) {
  try {
    const parsed = new URL(baseURL);
    return parsed.hostname || fallback;
  } catch {
    return fallback;
  }
}

export function formatProxyHost(host: string) {
  const trimmed = host.trim();
  if (!trimmed) return "";
  if (trimmed.startsWith("[") && trimmed.endsWith("]")) return trimmed;
  return trimmed.includes(":") ? `[${trimmed}]` : trimmed;
}

export function proxyURL(protocol: ProxyProtocol, host: string, port: number) {
  return `${protocol}://${formatProxyHost(host)}:${port}`;
}

export function quoteForCurl(value: string) {
  return `"${value.replace(/(["\\])/g, "\\$1")}"`;
}

export function proxyAuthArg(settings: Settings) {
  if (!settings.proxyUsername) return "";
  return `--proxy-user ${quoteForCurl(`${settings.proxyUsername}:${settings.proxyPassword || "PASSWORD"}`)} `;
}

export function proxyTestCommand(protocol: ProxyProtocol, host: string, port: number, settings: Settings) {
  return `curl -x ${proxyURL(protocol, host, port)} ${proxyAuthArg(settings)}https://api64.ipify.org`;
}
