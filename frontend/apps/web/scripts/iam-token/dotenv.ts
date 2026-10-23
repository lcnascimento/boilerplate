export function withEnvValue(
  content: string,
  key: string,
  value: string
): string {
  const lines = content.split("\n").filter((line) => line !== "");
  const entry = `${key}=${value}`;
  const index = lines.findIndex((line) => line.startsWith(`${key}=`));
  if (index === -1) lines.push(entry);
  else lines[index] = entry;
  return `${lines.join("\n")}\n`;
}
