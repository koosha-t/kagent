import React from "react";
import Image from "next/image";

export default function KAgentLogoWithText({ className }: { className?: string }) {
  return (
    <Image
      src="/kinagent.png"
      alt="Kinagent Logo"
      width={494}
      height={110}
      className={className}
      priority
    />
  );
}
