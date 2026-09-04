declare module "node:test" {
  type TestBody = () => void | Promise<void>;
  type Test = (name: string, body: TestBody) => void;

  const test: Test;
  export default test;
}

declare module "node:assert/strict" {
  interface StrictAssert {
    equal(actual: unknown, expected: unknown, message?: string | Error): void;
    deepEqual(
      actual: unknown,
      expected: unknown,
      message?: string | Error,
    ): void;
    ok(value: unknown, message?: string | Error): asserts value;
  }

  const assert: StrictAssert;
  export default assert;
}
