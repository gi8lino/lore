// Clipboard helpers shared by interactive controls.

export async function copyText(text: string): Promise<void> {
  if (!navigator.clipboard?.writeText) {
    throw new Error(
      "Clipboard API wird von diesem Browser oder Kontext nicht unterstützt.",
    );
  }

  await navigator.clipboard.writeText(text);
}
