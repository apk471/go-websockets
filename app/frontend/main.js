const { useMemo, useRef, useState } = React;

function App() {
  const [wsUrl, setWsUrl] = useState("ws://localhost:8080/ws");
  const [status, setStatus] = useState("disconnected");
  const [message, setMessage] = useState("");
  const [log, setLog] = useState([]);
  const socketRef = useRef(null);

  const isConnected = useMemo(() => status === "connected", [status]);

  const connect = () => {
    if (socketRef.current) {
      return;
    }

    setStatus("connecting");
    const socket = new WebSocket(wsUrl);
    socketRef.current = socket;

    socket.onopen = () => {
      setStatus("connected");
      setLog((prev) => [...prev, "Connected to server"]);
    };

    socket.onclose = () => {
      setStatus("disconnected");
      setLog((prev) => [...prev, "Disconnected"]);
      socketRef.current = null;
    };

    socket.onerror = () => {
      setStatus("error");
      setLog((prev) => [...prev, "WebSocket error"]);
    };

    socket.onmessage = (event) => {
      setLog((prev) => [...prev, `Server: ${event.data}`]);
    };
  };

  const disconnect = () => {
    if (socketRef.current) {
      socketRef.current.close();
      socketRef.current = null;
    }
  };

  const sendMessage = () => {
    if (!socketRef.current || socketRef.current.readyState !== WebSocket.OPEN) {
      setLog((prev) => [...prev, "Cannot send: socket is not open"]);
      return;
    }

    const value = message.trim();
    if (!value) {
      return;
    }

    socketRef.current.send(value);
    setLog((prev) => [...prev, `You: ${value}`]);
    setMessage("");
  };

  return React.createElement(
    "main",
    { className: "page" },
    React.createElement("h1", null, "React WebSocket Client"),
    React.createElement(
      "p",
      { className: "status" },
      `Status: ${status}`
    ),
    React.createElement(
      "div",
      { className: "controls" },
      React.createElement("input", {
        type: "text",
        value: wsUrl,
        onChange: (e) => setWsUrl(e.target.value),
        placeholder: "ws://localhost:8080/ws",
      }),
      React.createElement(
        "button",
        { onClick: connect, disabled: isConnected || status === "connecting" },
        "Connect"
      ),
      React.createElement(
        "button",
        { onClick: disconnect, disabled: !socketRef.current },
        "Disconnect"
      )
    ),
    React.createElement(
      "div",
      { className: "controls" },
      React.createElement("input", {
        type: "text",
        value: message,
        onChange: (e) => setMessage(e.target.value),
        placeholder: "Type a message",
        onKeyDown: (e) => {
          if (e.key === "Enter") {
            sendMessage();
          }
        },
      }),
      React.createElement(
        "button",
        { onClick: sendMessage, disabled: !isConnected },
        "Send"
      )
    ),
    React.createElement(
      "section",
      { className: "log" },
      React.createElement("h2", null, "Client Log"),
      React.createElement(
        "ul",
        null,
        log.map((entry, index) =>
          React.createElement("li", { key: `${entry}-${index}` }, entry)
        )
      )
    )
  );
}

const root = ReactDOM.createRoot(document.getElementById("root"));
root.render(React.createElement(App));
