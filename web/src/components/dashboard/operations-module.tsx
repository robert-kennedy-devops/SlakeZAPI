"use client";

import type { ReactNode } from "react";

export function OperationsModule({
  primary,
  sidebar,
}: {
  primary: ReactNode;
  sidebar: ReactNode;
}) {
  return (
    <section className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]" id="operations">
      <div className="grid gap-6">{primary}</div>
      <div className="grid gap-6">{sidebar}</div>
    </section>
  );
}
