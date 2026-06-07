

with open("main.tf", "a") as file:
    for start in range(1, 2000):
        result = f"""resource "tfcoremock_simple_resource" "name{start}" {{
integer = tfcoremock_simple_resource.name{start - 1}.integer + 1
}}\n"""
        file.write(result)


