type TruncatedTextProps = {
  children: string;
  className?: string;
};

export function TruncatedText({ children, className }: TruncatedTextProps) {
  const classes = ['truncated-text', className].filter(Boolean).join(' ');
  return (
    <span className={classes} title={children}>
      {children}
    </span>
  );
}
