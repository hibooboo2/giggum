#!/bin/bash

echo "🚀 Giggum Task Management Demo"
echo "================================="

echo ""
echo "1️⃣ Importing tasks from tasks.md..."
./ralph -task import

echo ""
echo "2️⃣ Listing all tasks..."
./ralph -task list

echo ""
echo "3️⃣ Showing task statistics..."
./ralph -task stats

echo ""
echo "✅ Demo completed! The SQLite database 'giggum_tasks.db' has been created with task data."