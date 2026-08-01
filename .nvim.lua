local dap = require("dap")

dap.adapters.go = {
	type = "server",
	port = 40000,
	host = "127.0.0.1",
}

dap.configurations.go = {
	{
		type = "go",
		name = "Attach Khazina (Docker Monorepo)",
		mode = "remote",
		request = "attach",
		substitutePath = {
			{
				from = vim.fn.getcwd() .. "/khazina",
				to = "/khazina",
			},
		},
	},
}

print("✅ Saftaja Monorepo Debug Config Loaded (Khazina)")
