"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";

import { DashboardClient } from "@/components/dashboard-client";
import { getAuthToken } from "@/lib/auth";

export default function DashboardPage() {
  const router = useRouter();

  useEffect(() => {
    if (!getAuthToken()) {
      router.replace("/login");
    }
  }, [router]);

  return <DashboardClient />;
}
