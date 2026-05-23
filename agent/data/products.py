from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class Product:
    id: str
    sku: str
    name: str
    description: str
    unit: str
    price_cents: int
    quantity: int
    category: str

    @property
    def price_inr(self) -> float:
        return self.price_cents / 100


def _slug(text: str) -> str:
    return "".join(c if c.isalnum() else "-" for c in text.upper()).strip("-")[:24]


def _build_catalog() -> list[Product]:
    """~180 dummy grocery SKUs across brands and categories."""
    items: list[tuple[str, str, str, str, str, int, int, str]] = []
    # name, description, unit, price_cents, stock, category  — brand in name

    def add(category: str, brand: str, product: str, unit: str, price: int, stock: int, desc: str = "") -> None:
        name = f"{brand} {product}".strip()
        items.append((name, desc or f"{brand} — {product}", unit, price, stock, category))

    # --- Cooking oil ---
    for brand in ("Fortune", "Saffola", "Dhara", "Patanjali", "Gemini", "Borges"):
        add("oil", brand, "Sunflower Oil", "1 litre", 14000 + hash(brand) % 2500, 70)
        add("oil", brand, "Mustard Oil", "1 litre", 15500 + hash(brand + "m") % 2000, 55)
        add("oil", brand, "Groundnut Oil", "1 litre", 17000 + hash(brand + "g") % 2000, 40)
    add("oil", "Fortune", "Sunflower Oil", "5 litre", 68000, 25, "Bulk pack")
    add("oil", "Saffola", "Gold Refined Oil", "1 litre", 19500, 60)
    add("oil", "Saffola", "Active Oil", "1 litre", 21000, 45)
    add("oil", "Dhara", "Rice Bran Oil", "1 litre", 15000, 50)
    add("oil", "Borges", "Extra Virgin Olive Oil", "250 ml", 45000, 20)

    # --- Rice ---
    for brand in ("India Gate", "Daawat", "Kohinoor", "Fortune", "Annapurna"):
        add("rice", brand, "Basmati Rice", "1 kg", 11000 + hash(brand) % 4000, 90)
        add("rice", brand, "Sona Masoori Rice", "5 kg", 32000 + hash(brand + "s") % 3000, 40)
    add("rice", "India Gate", "Brown Basmati", "1 kg", 16500, 35)
    add("rice", "Daawat", "Devaaya Basmati", "1 kg", 22000, 30)
    add("rice", "Kohinoor", "Super Basmati", "5 kg", 95000, 15)

    # --- Dal & pulses ---
    for brand in ("Tata Sampann", "Fortune", "Kohinoor", "24 Mantra"):
        add("dal", brand, "Toor Dal", "1 kg", 9000 + hash(brand) % 1500, 65)
        add("dal", brand, "Moong Dal", "500 g", 6500 + hash(brand + "mo") % 1000, 55)
        add("dal", brand, "Chana Dal", "1 kg", 8500 + hash(brand + "ch") % 1200, 50)
    add("dal", "Tata Sampann", "Masoor Dal", "1 kg", 7800, 60)
    add("dal", "Fortune", "Urad Dal", "500 g", 7200, 45)
    add("dal", "24 Mantra", "Organic Toor Dal", "1 kg", 14500, 25)

    # --- Dairy ---
    for brand, product, unit, price, stock in [
        ("Amul", "Taaza Toned Milk", "1 litre", 6200, 200),
        ("Amul", "Gold Full Cream Milk", "1 litre", 7200, 120),
        ("Amul", "Butter", "100 g", 5800, 80),
        ("Amul", "Pure Ghee", "500 ml", 32000, 35),
        ("Amul", "Paneer", "200 g", 9500, 50),
        ("Mother Dairy", "Toned Milk", "1 litre", 5800, 150),
        ("Mother Dairy", "Curd", "400 g", 3500, 90),
        ("Mother Dairy", "Classic Curd", "1 kg", 7200, 40),
        ("Nestle", "A+ Milk", "1 litre", 7800, 70),
        ("Gowardhan", "Ghee", "1 litre", 58000, 20),
        ("Britannia", "Cheese Slices", "200 g", 12500, 45),
        ("Epigamia", "Greek Yogurt", "85 g", 4500, 60),
    ]:
        add("dairy", brand, product, unit, price, stock)

    # --- Flour & grains ---
    for brand in ("Aashirvaad", "Pillsbury", "Fortune", "Patanjali"):
        add("flour", brand, "Whole Wheat Atta", "5 kg", 27000 + hash(brand) % 2000, 55)
        add("flour", brand, "Multigrain Atta", "1 kg", 6500 + hash(brand + "mg") % 800, 40)
    add("flour", "Aashirvaad", "Maida", "1 kg", 5200, 70)
    add("flour", "Fortune", "Besan", "500 g", 5800, 65)
    add("flour", "Pillsbury", "Chakki Fresh Atta", "10 kg", 48000, 20)

    # --- Essentials (salt, sugar, spices) ---
    for brand, product, unit, price in [
        ("Tata", "Salt", "1 kg", 2800),
        ("Tata", "Lite Salt", "1 kg", 4200),
        ("Madhur", "Sugar", "1 kg", 4500),
        ("Madhur", "Sugar", "5 kg", 21000),
        ("Everest", "Garam Masala", "50 g", 6500),
        ("Everest", "Chicken Masala", "100 g", 7200),
        ("Catch", "Red Chilli Powder", "200 g", 8500),
        ("Catch", "Turmeric Powder", "200 g", 6200),
        ("MDH", "Chana Masala", "100 g", 5800),
        ("MDH", "Dal Tadka Masala", "100 g", 5500),
        ("Keya", "Oregano", "8 g", 12000),
        ("Kellogg's", "Corn Flakes", "475 g", 18500),
    ]:
        add("essentials", brand, product, unit, price, 80 + hash(product) % 70)

    # --- Snacks ---
    snacks = [
        ("Maggi", "2-Minute Noodles Masala", "4 pack", 5600),
        ("Maggi", "Oats Noodles", "4 pack", 7200),
        ("Lays", "Classic Salted", "52 g", 2000),
        ("Lays", "Magic Masala", "82 g", 3500),
        ("Kurkure", "Masala Munch", "90 g", 2000),
        ("Haldiram", "Bhujia", "400 g", 8500),
        ("Haldiram", "Aloo Bhujia", "200 g", 4500),
        ("Parle-G", "Gold Biscuits", "1 kg", 12000),
        ("Oreo", "Original Biscuits", "120 g", 3500),
        ("Britannia", "Good Day Cookies", "600 g", 9500),
        ("Monaco", "Salted Biscuits", "400 g", 5500),
        ("Bingo", "Mad Angles", "66 g", 2000),
        ("Uncle Chips", "Spicy Treat", "55 g", 2000),
        ("Too Yumm", "Karela Chips", "50 g", 2500),
        ("Pringles", "Original", "107 g", 12500),
    ]
    for brand, product, unit, price in snacks:
        add("snacks", brand, product, unit, price, 100 + hash(brand + product) % 80)

    # --- Beverages ---
    beverages = [
        ("Tata Tea", "Gold", "500 g", 28000),
        ("Tata Tea", "Premium", "250 g", 15000),
        ("Brooke Bond", "Red Label", "500 g", 26000),
        ("Lipton", "Yellow Label Tea", "250 g", 14000),
        ("Nescafe", "Classic Coffee", "50 g", 18500),
        ("Bru", "Instant Coffee", "100 g", 32000),
        ("Real", "Mixed Fruit Juice", "1 litre", 11000),
        ("Tropicana", "Orange Juice", "1 litre", 12500),
        ("Coca-Cola", "Soft Drink", "750 ml", 4000),
        ("Pepsi", "Soft Drink", "750 ml", 4000),
        ("Sprite", "Lemon Drink", "750 ml", 4000),
        ("Frooti", "Mango Drink", "1 litre", 5500),
        ("Bournvita", "Health Drink", "500 g", 24500),
        ("Horlicks", "Classic Malt", "500 g", 26500),
        ("Complan", "Chocolate", "500 g", 32000),
    ]
    for brand, product, unit, price in beverages:
        add("beverages", brand, product, unit, price, 40 + hash(brand) % 50)

    # --- Fresh vegetables ---
    veggies = [
        ("Fresh", "Onion", "1 kg", 3500),
        ("Fresh", "Potato", "1 kg", 3000),
        ("Fresh", "Tomato", "1 kg", 4500),
        ("Fresh", "Green Capsicum", "500 g", 5500),
        ("Fresh", "Carrot", "500 g", 4000),
        ("Fresh", "Cauliflower", "1 piece", 3500),
        ("Fresh", "Spinach", "250 g", 2500),
        ("Fresh", "Coriander Bunch", "1 bunch", 1500),
        ("Fresh", "Ginger", "250 g", 3500),
        ("Fresh", "Garlic", "250 g", 4500),
        ("Fresh", "Lemon", "500 g", 3000),
        ("Fresh", "Cucumber", "500 g", 2500),
        ("Fresh", "Beans", "250 g", 4000),
        ("Fresh", "Brinjal", "500 g", 3500),
        ("Fresh", "Cabbage", "1 piece", 2800),
    ]
    for brand, product, unit, price in veggies:
        add("vegetables", brand, product, unit, price, 150 + hash(product) % 100)

    # --- Household ---
    household = [
        ("Surf Excel", "Matic Top Load", "1 kg", 32000),
        ("Surf Excel", "Easy Wash", "1 kg", 18000),
        ("Ariel", "Matic Front Load", "2 kg", 42000),
        ("Tide", "Plus Detergent", "1 kg", 12000),
        ("Rin", "Detergent Bar", "4 bars", 4500),
        ("Vim", "Dishwash Gel", "500 ml", 11000),
        ("Harpic", "Toilet Cleaner", "1 litre", 18500),
        ("Lizol", "Disinfectant", "500 ml", 12500),
        ("Colin", "Glass Cleaner", "500 ml", 9500),
        ("Odonil", "Room Freshener", "75 g", 6500),
        ("Good Knight", "Mosquito Repellent", "45 ml", 8500),
        ("All Out", "Machine Refill", "2 refills", 15000),
        ("Scotch-Brite", "Scrub Pad", "3 pcs", 5500),
        ("HIT", "Cockroach Spray", "200 ml", 14500),
    ]
    for brand, product, unit, price in household:
        add("household", brand, product, unit, price, 30 + hash(brand) % 40)

    # --- Personal care ---
    personal = [
        ("Dove", "Soap", "125 g", 7500),
        ("Lux", "Soft Touch Soap", "150 g", 4500),
        ("Lifebuoy", "Total Soap", "4 pcs", 12000),
        ("Colgate", "Strong Teeth", "200 g", 11500),
        ("Sensodyne", "Toothpaste", "70 g", 18500),
        ("Dettol", "Handwash", "200 ml", 9900),
        ("Dettol", "Antiseptic Liquid", "500 ml", 18500),
        ("Pantene", "Shampoo", "340 ml", 32000),
        ("Head & Shoulders", "Anti-Dandruff", "340 ml", 34000),
        ("Parachute", "Coconut Oil", "200 ml", 8500),
        ("Nivea", "Soft Cream", "100 ml", 12500),
        ("Gillette", "Shaving Foam", "196 g", 22000),
        ("Whisper", "Ultra XL", "7 pads", 9500),
        ("Stayfree", "Secure Cotton", "6 pads", 7500),
    ]
    for brand, product, unit, price in personal:
        add("personal_care", brand, product, unit, price, 45 + hash(brand + product) % 55)

    # --- Bakery ---
    bakery = [
        ("Britannia", "Milk Bread", "400 g", 4500),
        ("Britannia", "Brown Bread", "400 g", 5000),
        ("Harvest Gold", "White Bread", "400 g", 4200),
        ("Britannia", "Fruit Cake", "150 g", 3500),
        ("Britannia", "Rusk", "300 g", 5500),
        ("English Oven", "Pav", "6 pcs", 3000),
        ("Modern", "Sandwich Bread", "400 g", 4000),
    ]
    for brand, product, unit, price in bakery:
        add("bakery", brand, product, unit, price, 60 + hash(product) % 40)

    # --- Eggs & frozen ---
    add("dairy", "Farm Fresh", "Brown Eggs", "6 pieces", 7200, 60)
    add("dairy", "Farm Fresh", "Brown Eggs", "12 pieces", 13800, 40)
    add("snacks", "McCain", "French Fries", "420 g", 14500, 25)
    add("snacks", "Yummiez", "Veg Nuggets", "400 g", 16500, 20)

    catalog: list[Product] = []
    for i, (name, desc, unit, price, stock, category) in enumerate(items, start=1):
        pid = f"p{i}"
        sku = _slug(f"{category}-{name}-{unit}")[:32]
        catalog.append(
            Product(
                id=pid,
                sku=sku,
                name=name,
                description=desc,
                unit=unit,
                price_cents=price,
                quantity=stock,
                category=category,
            )
        )
    return catalog


CATALOG: list[Product] = _build_catalog()


def get_product(product_id: str) -> Product | None:
    return next((p for p in CATALOG if p.id == product_id), None)


def format_product_line(p: Product) -> str:
    return f"• {p.name} ({p.unit}) — ₹{p.price_inr:.2f} | stock: {p.quantity} | id: {p.id}"
