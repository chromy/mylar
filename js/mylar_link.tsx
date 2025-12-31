import { Link } from "wouter";
import { ComponentProps } from "react";

interface MylarLinkProps extends ComponentProps<typeof Link> {}

export const MylarLink = ({ className = "", ...props }: MylarLinkProps) => {
  return (
    <Link
      className={`text-blue-600 hover:text-blue-800 underline hover:no-underline transition-colors ${className}`}
      {...props}
    />
  );
};
