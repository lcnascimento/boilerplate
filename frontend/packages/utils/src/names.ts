/**
 * Up to two uppercase initials for an avatar: first and last word, or the
 * first two letters of a single word. Returns "?" for a blank name.
 */
export function getInitials(name: string): string {
  const words = name.split(" ").filter(Boolean);
  const first = words[0];
  if (first === undefined) return "?";

  const last = words.at(-1);
  if (words.length === 1 || last === undefined) {
    return first.slice(0, 2).toUpperCase();
  }

  return (first.charAt(0) + last.charAt(0)).toUpperCase();
}
