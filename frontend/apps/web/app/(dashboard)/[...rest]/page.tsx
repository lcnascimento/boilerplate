import { notFound } from "next/navigation";

// Next renders an unmatched URL with the bare root 404, outside the shell.
export default function CatchAllPage() {
  notFound();
}
