from google.adk.agents import Agent
from google.adk.tools import FunctionTool

from tools import train_tools

INSTRUCTION = """
You are a help assistant for *train food on WhatsApp*. Most users order via buttons;
you only answer questions (PNR kya hai, delivery kaise hoti hai, etc.).

If they want to order, tell them:
1. Send 10-digit PNR
2. Choose delivery station from the list (vendors shown per stop)
3. Choose vendor → menu → cart → place order

Demo PNR: 1234567890. Reply in Hindi/Hinglish/English. Keep answers short.
Do not invent prices or stock — use tools only if they give a PNR and ask about their trip.
"""

root_agent = Agent(
    model="gemini-2.5-flash",
    name="whatsapp_train_food_agent",
    description="Train food ordering on WhatsApp — PNR, station delivery, vendor menu, checkout.",
    instruction=INSTRUCTION,
    tools=[
        FunctionTool(func=train_tools.get_stops_with_vendors),
        FunctionTool(func=train_tools.lookup_pnr),
        FunctionTool(func=train_tools.select_delivery_station),
        FunctionTool(func=train_tools.list_vendors_at_station),
        FunctionTool(func=train_tools.select_vendor),
        FunctionTool(func=train_tools.browse_menu),
        FunctionTool(func=train_tools.add_meal_to_cart),
        FunctionTool(func=train_tools.view_train_cart),
        FunctionTool(func=train_tools.remove_from_train_cart),
        FunctionTool(func=train_tools.clear_train_cart),
        FunctionTool(func=train_tools.place_train_order),
        FunctionTool(func=train_tools.get_train_order_status),
        FunctionTool(func=train_tools.get_recent_orders),
    ],
)
