export default function (pi) {
	pi.registerCommand("ahweb-rewind", {
		description: "Rewind a Pi session to an entry",
		handler: async (entryID, ctx) => {
			entryID = entryID.trim();
			if (!entryID) throw new Error("entry ID is required");
			const result = await ctx.navigateTree(entryID);
			if (result.cancelled) throw new Error("Pi rewind was cancelled");
		},
	});
}
