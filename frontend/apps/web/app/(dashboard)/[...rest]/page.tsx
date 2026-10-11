import { notFound } from "next/navigation";

// An unmatched URL has no page; sending it here renders the group's
// not-found inside the shell instead of the bare root 404.
export default function CatchAllPage() {
  notFound();
}
