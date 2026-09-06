// DOM helpers for markup that is required by a component template contract.

export function requiredElement<ElementType extends Element>(
  root: ParentNode,
  selector: string,
): ElementType {
  const element = root.querySelector<ElementType>(selector);
  if (!element) throw new Error(`Missing required element: ${selector}`);

  return element;
}
