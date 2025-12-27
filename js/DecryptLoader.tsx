import { useState, useEffect } from "react";

export const DecryptLoader = () => {
  const [text, setText] = useState("LOADING");
  const originalText = "LOADING";
  const chars = "{}?0XYZ#@![]";

  useEffect(() => {
    let iteration = 0;
    const interval = setInterval(() => {
      setText(prev =>
        prev
          .split("")
          .map((letter, index) => {
            if (index < Math.floor(iteration)) {
              return originalText[index];
            }
            return chars[Math.floor(Math.random() * chars.length)];
          })
          .join(""),
      );

      if (iteration >= originalText.length + 3) {
        iteration = 0; // Loop the effect
      }
      iteration += 1 / 3;
    }, 50);

    return () => clearInterval(interval);
  }, []);

  return (
    <div className="text-center m-2 font-mono text-green-500 text-xl font-bold">
      {text}
    </div>
  );
};
