import { clsx, type ClassValue } from "clsx";

export function cn(...inputs: ClassValue[]) {
  return clsx(inputs);
}

export function formatDate(value?: string) {
  if (!value) return "-";
  const date = new Date(value);
  return new Intl.DateTimeFormat("pt-BR", {
    dateStyle: "short",
    timeStyle: "short",
  }).format(date);
}

export function shorten(value: string, left = 6, right = 4) {
  if (value.length <= left + right + 3) return value;
  return `${value.slice(0, left)}...${value.slice(-right)}`;
}
