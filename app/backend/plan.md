So basically what i want to implement is a websocket implementation where in one Match event gets trigger to many users without them refreshing i.e in short FAN OUT in realtime

First let us do something like:

1. create a attach websocketserver which uses the same port as of our REST api server but with a path of '/ws' all the request will be using this websocketserver

2. Let us just send a json message as "welcome" to all our connected clients first (whenever they connect first they should get a welcome message)

3. Make changes to the main.go HOST PORT then go into the list handler where we create the matches and hook that with this WS to broadcast all the matches in realtime to all the clients connected (so basically when we hit the create match endpoint with any of our WS client connected as soon as i hit the endpoint the create match should reflect there)
