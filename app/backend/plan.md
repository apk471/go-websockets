1. i need to create a commentary router group with /matches/:id/commentary as one group 
2. Create schemas with the below schemas also with a new db table called as commentary
export const listCommentaryQuerySchema = z.object({
  limit: z.coerce.number().int().positive().max(100).optional(),
});

export const createCommentarySchema = z.object({
  minute: z.number().int().nonnegative(),
  sequence: z.number().int().optional(),
  period: z.string().optional(),
  eventType: z.string().optional(),
  actor: z.string().optional(),
  team: z.string().optional(),
  message: z.string().min(1),
  metadata: z.record(z.string(), z.any()).optional(),
  tags: z.array(z.string()).optional(),
});

3. create two main routes both in the commentary group GET / route and the POST / route

GET / -> 
1. Validate req.params using matchldParamSchema and req.query using
listCommentaryQuerySchema.
2. Fetch data from the "commentary" table where "matchid" equals the ID from
params.
3. Order the results by "createdAt" in descending order so the newest events
appear first.
4. Apply a limit based on the query parameter (defaulting to 100 with a
MAX_LIMIT safety cap).
5. Use ES Modules and handle errors with try/catch


POST / -> this is used to create a commetary of a particular match
1. validate req params using matchldParamSchema and req.body using
createCommentaryShcmea. insert the data into the commentary table and
return the result

