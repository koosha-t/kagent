import React from "react";
import Image from "next/image";

export default function KagentLogo({ className }: { className?: string }) {
  return (
    <Image
      src="/kinagent.png"
      alt="Kinagent Logo"
      width={378}
      height={286}
      className={className}
      priority
    />
  );
}
