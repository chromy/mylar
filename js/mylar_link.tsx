import { Link } from "wouter";
import { type ReactNode } from "react";

interface MylarLinkProps {
  href: string;
  children?: ReactNode;
  className?: string;
}

export const MylarLink = ({ className = "", children, href }: MylarLinkProps) => {
  return (
    <Link
      href={href}
      className={`text-blue-600 hover:text-blue-800 underline hover:no-underline transition-colors ${className}`}
    >
      {children}
    </Link>
  );
};
